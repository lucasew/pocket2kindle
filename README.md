# pocket2kindle
- A thing to convert pocket articles to kindle books and send them to your kindle email
- If something is missing it will say you (environment variables, softwares in PATH)

## Dependencies
- Build
  - Go
- Run
  - Calibre: `ebook-convert` command
  - The authentication tokens as environment variables for pocket and for SMTP server if you specified a destination email
    - **TIP**: Use a secondary google account because you will need to enable access through insecure applications if using gmail
    - **TIP**: [Dotenv](https://github.com/lucasew/dotenv/)
  
# The flow
- Fetch n*3 articles from pocket
- Parse n articles using a very competent port of Readability for Golang
- Generate a EPUB from the parsed articles with favorite links pointing to pocket API
- Convert that EPUB to a MOBI file
- Send the MOBI file to the provided email via SMTP server (the origin address must be authorized in your amazon account)
- Wait a few moments while the amazon digital gnomes deliver the file to your device
- Remove that intermediate epub file if you didn't specify it to not remove
- Profit!

# Advantages
- It's free
- You have the exact notion of what stage is your delivery

# Disadvantages
- Setup authentication to a SMTP server (only done once)
- Setup authentication via the oauth flow for pocket (only done once too)

# More information
Use the --help flag of the program

Ex:
- `go run ./cmd/p2k --help` or if in `$PATH`
- `p2k --help`
