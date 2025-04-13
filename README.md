# url-shortener
A basic URL shortening service

This is a hobby project to learn Golang. The application starts web server that takes a long URL as input and returns a shortened version of it. The shortened URL can then be used to redirect to the original long URL. The application uses an in-memory store to map shortened URLs to their original counterparts.

## Steps to run the application
> 1. Clone the repository
> 2. Change directory to the project folder
> 3. Execute the command `go mod tidy` to download the dependencies
> 4. Run the application using `go run main.go`

System requirements:
1. A Login page. User is required to provide email to login. An OTP will be sent to the email.
2. User profile page where user can provide name.
3. A shortened URL list page.
4. A URL shortening side drawer where user can provide long URL and get a shortened URL. On successful shortening, the shortened URL will be copied to clipboard.
5. On the list page, user can edit/delete the shortened URL.

Entities:
1. User
   - id
   - email
   - name
   - created_at
   - updated_at
2. User OTP (Will be deleted on successful login)
   - user_id
   - otp
   - valid_until
3. Shortened URL
   - id
   - user_id
   - long_url
   - short_url
   - created_at
   - updated_at

Roadmap:
1. Use config file for:
    - Port
    - URL shortening base URL
2. Database implementation
    - Postgres
3. Cache implementation
    - Redis
    - In-memory
4. Add OTP based login for user
5. Create JWT token on successful login
6. JWT token based authentication for all the APIs
7. Add a frontend admin panel
5. Add docker deployment
6. Add unit tests
7. Add integration tests
8. Add logging
9. Add metrics