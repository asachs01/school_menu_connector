// Simple in-memory rate limiter: tracks request counts per IP per minute.
const rateLimitMap = new Map();
const RATE_LIMIT_WINDOW_MS = 60_000;
const RATE_LIMIT_MAX = 30;

function isRateLimited(ip) {
  const now = Date.now();
  const entry = rateLimitMap.get(ip);

  if (!entry || now - entry.windowStart > RATE_LIMIT_WINDOW_MS) {
    rateLimitMap.set(ip, { windowStart: now, count: 1 });
    return false;
  }

  entry.count++;
  if (entry.count > RATE_LIMIT_MAX) {
    return true;
  }
  return false;
}

// Periodically prune stale entries to avoid unbounded growth.
function pruneRateLimitMap() {
  const now = Date.now();
  for (const [ip, entry] of rateLimitMap) {
    if (now - entry.windowStart > RATE_LIMIT_WINDOW_MS * 2) {
      rateLimitMap.delete(ip);
    }
  }
}

function corsHeaders(origin, allowedOrigin) {
  // Allow the configured origin and also localhost for development.
  const allowed =
    origin === allowedOrigin || origin === "http://localhost:8080";
  return {
    "Access-Control-Allow-Origin": allowed ? origin : allowedOrigin,
    "Access-Control-Allow-Methods": "GET, OPTIONS",
    "Access-Control-Allow-Headers": "Content-Type, X-Auth-Token",
    "Access-Control-Max-Age": "86400",
  };
}

// Check if the request is authenticated via auth token (server-to-server).
function isAuthenticated(request, env) {
  const token = request.headers.get("X-Auth-Token");
  return token && env.AUTH_TOKEN && token === env.AUTH_TOKEN;
}

export default {
  async fetch(request, env, ctx) {
    // Prune rate limit map periodically.
    pruneRateLimitMap();

    const url = new URL(request.url);
    const allowedOrigin = env.ALLOWED_ORIGIN || "https://schoolmenuconnector.com";
    const origin = request.headers.get("Origin") || "";
    const authed = isAuthenticated(request, env);

    // Handle CORS preflight.
    if (request.method === "OPTIONS") {
      return new Response(null, {
        status: 204,
        headers: corsHeaders(origin, allowedOrigin),
      });
    }

    // Only allow GET requests.
    if (request.method !== "GET") {
      return new Response(JSON.stringify({ error: "Method not allowed" }), {
        status: 405,
        headers: { "Content-Type": "application/json" },
      });
    }

    // Only proxy the FamilyMenu endpoint.
    if (url.pathname !== "/api/FamilyMenu") {
      return new Response(JSON.stringify({ error: "Not found" }), {
        status: 404,
        headers: { "Content-Type": "application/json" },
      });
    }

    // Validate required query params.
    const required = ["buildingId", "districtId", "startDate", "endDate"];
    const missing = required.filter((p) => !url.searchParams.get(p));
    if (missing.length > 0) {
      return new Response(
        JSON.stringify({ error: `Missing required params: ${missing.join(", ")}` }),
        {
          status: 400,
          headers: { "Content-Type": "application/json" },
        }
      );
    }

    // For non-authenticated requests, enforce CORS origin and rate limiting.
    if (!authed) {
      const allowed =
        origin === allowedOrigin || origin === "http://localhost:8080";
      if (!allowed) {
        return new Response(JSON.stringify({ error: "Forbidden" }), {
          status: 403,
          headers: { "Content-Type": "application/json" },
        });
      }

      // Rate limiting by client IP.
      const clientIP =
        request.headers.get("CF-Connecting-IP") || "unknown";
      if (isRateLimited(clientIP)) {
        return new Response(JSON.stringify({ error: "Rate limit exceeded" }), {
          status: 429,
          headers: {
            "Content-Type": "application/json",
            "Retry-After": "60",
          },
        });
      }
    }

    // Build upstream URL.
    const upstreamURL = `https://api.linqconnect.com/api/FamilyMenu?${url.searchParams.toString()}`;

    // Check Cloudflare cache first.
    const cache = caches.default;
    const cacheKey = new Request(upstreamURL, request);
    let cachedResponse = await cache.match(cacheKey);
    if (cachedResponse) {
      // Return cached response with CORS headers.
      const headers = new Headers(cachedResponse.headers);
      for (const [k, v] of Object.entries(corsHeaders(origin, allowedOrigin))) {
        headers.set(k, v);
      }
      return new Response(cachedResponse.body, {
        status: cachedResponse.status,
        headers,
      });
    }

    // Fetch from upstream with browser-like headers.
    const upstreamResponse = await fetch(upstreamURL, {
      method: "GET",
      headers: {
        "User-Agent":
          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
        Accept: "application/json, text/plain, */*",
        "Accept-Language": "en-US,en;q=0.9",
        Origin: "https://linqconnect.com",
        Referer: "https://linqconnect.com/",
      },
    });

    if (!upstreamResponse.ok) {
      return new Response(
        JSON.stringify({
          error: "Upstream API error",
          status: upstreamResponse.status,
        }),
        {
          status: upstreamResponse.status,
          headers: {
            "Content-Type": "application/json",
            ...corsHeaders(origin, allowedOrigin),
          },
        }
      );
    }

    const body = await upstreamResponse.text();

    // Build cacheable response (6 hour TTL).
    const responseHeaders = {
      "Content-Type": "application/json",
      "Cache-Control": "public, max-age=21600",
      ...corsHeaders(origin, allowedOrigin),
    };

    const response = new Response(body, {
      status: 200,
      headers: responseHeaders,
    });

    // Store in Cloudflare cache (non-blocking).
    ctx.waitUntil(cache.put(cacheKey, response.clone()));

    return response;
  },
};
