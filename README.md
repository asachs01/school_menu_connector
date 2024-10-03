![Release](https://github.com/asachs01/school_menu_connector/actions/workflows/release.yml/badge.svg)
<a href="https://www.buymeacoffee.com/aaronsachs" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/v2/arial-yellow.png" alt="Buy Me A Coffee" width="120" height="30" ></a>

# School Menu Connector

School Menu Connector is a Go application that fetches school lunch menus from the LINQ Connect API and sends them via email. It's designed to be flexible, allowing configuration through both command-line flags and environment variables.

## Features

- Fetch lunch menus for a specific date or date range
- Send menus via email
- Configurable through command-line flags or environment variables
- Customizable email subject
- Generate ICS files via web API endpoint

## Web API

While there is a CLI tool that you can use to send emails and generate ICS files, School Menu Connector now provides a web API for generating ICS files:

### API Endpoint

Send a POST request to `/get-menu` with the following form data:

- `buildingId`: The ID of the school building
- `districtId`: The ID of the school district
- `startDate`: Start date for the menu (format: MM-DD-YYYY)
- `endDate`: End date for the menu (format: MM-DD-YYYY)

Example using curl:

```
curl -X POST https://localhost:8080/get-menu \
     -F "buildingId=YOUR_BUILDING_ID" \
     -F "districtId=YOUR_DISTRICT_ID" \
     -F "startDate=MM-DD-YYYY" \
     -F "endDate=MM-DD-YYYY"
```

## Prerequisites

The biggest thing that you need is the building and district IDs.  To find them:

1. Go to https://linqconnect.com/ and sign in
2. Click on the hamburger menu in the top left corner
3. Click on "School Menu"
4. Click on your school
5. Open up the browser's developer tools (usually F12)
6. Click on the "Network" tab
7. Search for "FamilyMenu"
8. Click on the request
9. In the "Query String Parameters" section, find the `districtId` parameter and the `buildingId` parameter
10. Copy the values and save them in a secure location

## Running the notifier

Head to the [Releases](https://github.com/asachs01/school_menu_connector/releases) page to download a pre-built binary for your platform, or build from source.

### Linux/MacOS
Depending on your platform, you may need to make the binary executable:

```
chmod +x school_menu_connector
```

You can then either set up a cron job to run the notifier at a regular interval, or run it manually with the following command:

```
./school_menu_connector -building=YOUR_BUILDING_ID -district=YOUR_DISTRICT_ID -recipient=recipient@example.com -sender=your_email@example.com -password=your_email_password -smtp=smtp.example.com:587 -subject="School Lunch Menu" -start=MM-DD-YYYY -end=MM-DD-YYYY
```

### Windows

For Windows, you can set up a scheduled task to run the notifier at a regular interval, or run it manually with the following command:

```
./school_menu_connector.exe -building=YOUR_BUILDING_ID -district=YOUR_DISTRICT_ID -recipient=recipient@example.com -sender=your_email@example.com -password=your_email_password -smtp=smtp.example.com:587 -subject="School Lunch Menu" -start=MM-DD-YYYY -end=MM-DD-YYYY
```

## Prerequisites (for building from source)

- Go 1.16 or later
- Access to a SMTP server for sending emails

## Installation

1. Clone the repository:
   ```
   git clone https://github.com/yourusername/school_menu_connector.git
   cd school_menu_connector
   ```

2. Build the application:
   ```
   go build -o school_menu_connector
   ```

## Usage

You can run the application using command-line flags or environment variables.

### Command-line flags:

```
./school_menu_connector \
-building=YOUR_BUILDING_ID \
-district=YOUR_DISTRICT_ID \
-recipient=recipient@example.com \
-sender=your_email@example.com \
-password=your_email_password \
-smtp=smtp.example.com:587 \
-subject="School Lunch Menu" \
-startDate=MM-DD-YYYY \
-endDate=MM-DD-YYYY \
-email \
-ics \
-week-start=MM-DD-YYYY \
-ics-output-path=path/to/output.ics \
-debug
```

### Environment variables:

```
export BUILDING_ID=YOUR_BUILDING_ID
export DISTRICT_ID=YOUR_DISTRICT_ID
export RECIPIENT_EMAIL=recipient@example.com
export SENDER_EMAIL=your_email@example.com
export EMAIL_PASSWORD=your_email_password
export SMTP_SERVER=smtp.example.com:587
export EMAIL_SUBJECT="School Lunch Menu"
export START_DATE=MM-DD-YYYY
export END_DATE=MM-DD-YYYY
export WEEK_START=MM-DD-YYYY
export ICS_OUTPUT_PATH=/path/to/output.ics
export DEBUG=true
```

## Configuration Options

- `building`: Building ID (required)
- `district`: District ID (required)
- `recipient`: Recipient email address (required for email)
- `sender`: Sender email address (required for email)
- `password`: Sender email password (required for email)
- `smtp`: SMTP server and port (default: smtp.gmail.com:587)
- `subject`: Email subject line (default: "Lunch Menu")
- `startDate`: Start date for menu range (format: MM-DD-YYYY, default: today)
- `endDate`: End date for menu range (format: MM-DD-YYYY, default: same as start date)
- `email`: Flag to enable email sending
- `ics`: Flag to enable ICS file generation
- `week-start`: Start date of the week for ICS file (format: MM-DD-YYYY, default: startDate)
- `ics-output-path`: Custom path for the ICS file output
- `debug`: Enable debug output for troubleshooting

## Examples
### Sending an email

Using a cronjob to set up notifications for the next day is easy enough. For me, that means that I want to get a notification every night Sunday-Thursday so that I can give the info to my kid:

```shell
0 17 * * 0-4 /home/myuser/.bin/school_menu_connector
```

The equivalent in Windows would be dropping the executable in a location you want to run it from and creating a scheduled task with something like:

```
schtasks /create /tn "School Menu Connector Notifications" /tr "C:\path\to\your\binary.exe" /sc weekly /d SUN,MON,TUE,WED,THU /st 17:00
```

### Generating an ICS file

If you want to use the ICS file to display the lunch menu in a web browser or other calendar application, you can use the following command:

```shell
./school_menu_connector -building=YOUR_BUILDING_ID -district=YOUR_DISTRICT_ID -ics -week-start=MM-DD-YYYY
```

Optionally, you can specify an output path for the ICS file:

```shell
./school_menu_connector -building=YOUR_BUILDING_ID -district=YOUR_DISTRICT_ID -ics -week-start=MM-DD-YYYY -ics-output-path=path/to/output.ics
```

## Notes

- If using Gmail as your SMTP server, you may need to use an "App Password" instead of your regular password and enable "Less secure app access" in your Google Account settings.
- Handle email credentials securely and avoid committing them to version control.

## Deployment

When deploying to DigitalOcean App Platform:

1. Ensure that the "Preserve Path Prefix" option is checked for your routes in the App Platform configuration.
2. This setting allows the full request path to be passed to your application, which is crucial for proper routing.

## License

[GPLv3](LICENSE)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Thanks and Credits

Thank you to @evanhsu for his work on the [Magic Mirror module](https://github.com/evanhsu/MMM-TitanSchoolMealMenu/tree/main) that inspired this project
