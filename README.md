# LINQ Connect Menu Notifier

LINQ Connect Menu Notifier is a Go application that fetches school lunch menus from the LINQ Connect API and sends them via email. It's designed to be flexible, allowing configuration through both command-line flags and environment variables.

## Features

- Fetch lunch menus for a specific date or date range
- Send menus via email
- Configurable through command-line flags or environment variables
- Customizable email subject

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

Head to the [Releases](https://github.com/aaronsachs/linqconnect-menu-notifier/releases) page to download a pre-built binary for your platform, or build from source.

### Linux/MacOS
Depending on your platform, you may need to make the binary executable:

```
chmod +x linq_connect_menu_notifier
```

You can then either set up a cron job to run the notifier at a regular interval, or run it manually with the following command:

```
./linq_connect_menu_notifier -building=YOUR_BUILDING_ID -district=YOUR_DISTRICT_ID -recipient=recipient@example.com -sender=your_email@example.com -password=your_email_password -smtp=smtp.example.com:587 -subject="School Lunch Menu" -start=MM-DD-YYYY -end=MM-DD-YYYY
```

### Windows

For Windows, you can set up a scheduled task to run the notifier at a regular interval, or run it manually with the following command:

```
./linq_connect_menu_notifier.exe -building=YOUR_BUILDING_ID -district=YOUR_DISTRICT_ID -recipient=recipient@example.com -sender=your_email@example.com -password=your_email_password -smtp=smtp.example.com:587 -subject="School Lunch Menu" -start=MM-DD-YYYY -end=MM-DD-YYYY
```

## Prerequisites (for building from source)

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
   go build -o linq_connect_menu_notifier
   ```

## Usage

You can run the application using command-line flags or environment variables.

### Command-line flags:

```
./linq_connect_menu_notifier \
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

## Examples

Using a cronjob to set up notifications for the next day is easy enough. For me, that means that I want to get a notification every night Sunday-Thursday so that I can give the info to my kid:

```shell
0 17 * * 0-4 /home/myuser/.bin/linq_connect_menu_notifier
```

The equivalent in Windows would be dropping the executable in a location you want to run it from and creating a scheduled task with something like:

```
schtasks /create /tn "LINQConnect Menu Notifications" /tr "C:\path\to\your\binary.exe" /sc weekly /d SUN,MON,TUE,WED,THU /st 17:00
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
