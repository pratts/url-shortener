# url-shortener
A basic URL shortening service

This is a hobby project to learn Golang. The application starts web server that takes a long URL as input and returns a shortened version of it. The shortened URL can then be used to redirect to the original long URL. The application uses an in-memory store to map shortened URLs to their original counterparts.

## Steps to run the application
> 1. Clone the repository
> 2. Change directory to the project folder
> 3. Execute the command `go mod tidy` to download the dependencies
> 4. Run the application using `go run main.go`