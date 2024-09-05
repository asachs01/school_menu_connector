# LinqConnect Menu Notifier

LinqConnect Menu Notifier is a Go application that fetches school lunch menus from the LinqConnect API and sends them via email. It's designed to be flexible, allowing configuration through both command-line flags and environment variables.

## Features

- Fetch lunch menus for a specific date or date range
- Send menus via email
- Configurable through command-line flags or environment variables
- Customizable email subject

## Prerequisites

- Go 1.16 or later
- Access to a SMTP server for sending emails

## Installation

1. Clone the repository:
   ```
   git clone https://github.com/yourusername/linqconnect-menu-notifier.git
   cd linqconnect-menu-notifier
   ```

2. Build the application:
   ```
   go build -o linqconnect_menu_notifier
   ```

## Usage

You can run the application using command-line flags or environment variables.

### Command-line flags:

```
./linqconnect_menu_notifier \
-building=YOUR_BUILDING_ID \
-district=YOUR_DISTRICT_ID \
-recipient=recipient@example.com \
-sender=your_email@example.com \
-password=your_email_password \
-smtp=smtp.example.com:587 \
-subject="School Lunch Menu" \
-start=MM-DD-YYYY \
-end=MM-DD-YYYY
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
```

## Configuration Options

- `building`: Building ID (required)
- `district`: District ID (required)
- `recipient`: Recipient email address (required)
- `sender`: Sender email address (required)
- `password`: Sender email password (required)
- `smtp`: SMTP server and port (default: smtp.gmail.com:587)
- `subject`: Email subject line (default: "Lunch Menu")
- `start`: Start date for menu range (format: MM-DD-YYYY, default: today)
- `end`: End date for menu range (format: MM-DD-YYYY, default: same as start date)

Then run the application:

```
./linqconnect_menu_notifier
```

## Notes

- If using Gmail as your SMTP server, you may need to use an "App Password" instead of your regular password and enable "Less secure app access" in your Google Account settings.
- Handle email credentials securely and avoid committing them to version control.

## License

[MIT License](LICENSE)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Thanks and Credits

Thank you to @evanhsu for his work on the [Magic Mirror module](https://github.com/evanhsu/MMM-TitanSchoolMealMenu/tree/main) that inspired this project
