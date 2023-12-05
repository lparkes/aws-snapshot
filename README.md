Compile the code with "go build ./main.go". I used Go 1.21, but I did
test that it compiles with Go 1.18.

Upload a local file into a new AWS EC2 snapshot with the command
"./main FILENAME".


CLI options are:

  -d string
    	Description for the snapshot being created (default "NetBSD AMI")
  -debug
    	Enable debug logging
  -w int
    	Number of concurrent writers (default 100)

I recommend turning on debug logging because that provides some sense
of upload progress.
