# pocket2kindle
- A thing to convert pocket articles to kindle books and send them to your kindle email
- If something is missing it will say you (environment variables, softwares in PATH)

## Dependencies
- Build
  - Go
- Run
  - Calibre: `ebook-convert` command
  - The authentication tokens as environment variables for pocket and for SMTP server if you specified a destination email
    - **TIP**: Use a secondary google account because you will need to enable access through insecure applications if using gmail. GMX is also pretty convenient to use this.
    - **TIP**: [Dotenv](https://github.com/lucasew/dotenv/)
  
# The flow
- There is a background job getting a stream of articles from new to old
- Another background job parses the articles in memory skipping the ones that aren't parseable until it gets the desired amount
- After article parsing the book starts being assembled
- Each article before being added to the book have its images downloaded and added to the book fixing the image references in the HTML
- After all the articles are stored in the epub file the file is packed.
- After the epub packaging `ebook-convert` is started to convert it to mobi (this is required because I didn't found good libraries to create mobi directly)
- With the mobi file created the epub file is marked as a intermediate file and will be deleted unless you pass `-d`
- If the email destination is specified the mobi file is sent using the SMTP server provided. If the mail is successfully sent, the mobi file is marked as a intermediate file.
- Internediate files are deleted if `-d` is not provided.
- Profit!

# Advantages
- It's free
- You have the exact notion of what stage is your delivery (lots of logs)

# Disadvantages
- Setup authentication to a SMTP server (only done once)
- Setup authentication via the oauth flow for pocket (only done once too)

# More information
Use the --help flag of the program

Ex:
- `go run ./cmd/p2k --help` or if in `$PATH`
- `p2k --help`
