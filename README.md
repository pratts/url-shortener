# url-shortener
A basic URL shortening service

This is a hobby project to learn Golang. The application has 2 components:
1. Admin Service: This is a RESTful API server that allows users to shorten URLs. It provides 
    - A user interface for users to input long URLs and receive shortened versions.
    - A list of previously shortened URLs.
2. Url Redirector: This is a simple HTTP server that redirects requests for shortened URLs to their original long URLs.

## Project Structure
```
url-shortener/
├── auth/               # JWT Authentication token validation and creation related code
├── cache/              # Redis Cache related code
├── configs/            # Configuration files. Base server, database, cache, etc.
├── db/                 # Postgres database object
├── models/             # Database models for URLs, Users and URL Redirection
├── server/             # Separate start points for admin and redirector services
├── urls/               # URL shortener related code for admin and redirector services
├── users/              # User related code for admin service
```

## Getting Started
### Prerequisites
- Go 1.24.1 or later
- Postgres 15 or later
- Redis 7.4 or later
- Docker (optional, for deployment)
- Docker Compose (optional, for deployment)
- Make sure you have the necessary permissions to run the application and access the database.

### Installation
1. Clone the repository:
   ```bash
   git clone https://github.com/pratts/url-shortener.git
   cd url-shortener
   ```
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Set up the database:
   - Create a Postgres database and user for the application.
4. Set up Redis:
   - Install Redis and start the server.
5. Configure the application:
   - Update the .env.example file with server, database and cache configurations
   - Rename the file to .env or create a new .env file with the same name and copy the parameters from 
   .env.example
6. Run the applications on terminal:
    - Start the admin service:
    ```bash
    go run server/admin/main.go
    ```
    - Start the redirector service:
    ```bash
    go run server/redirector/main.go
    ```
    - Alternatively, you can use Docker to run the application:
    ```bash
    docker-compose up --build
    ```
    - This will start the application and expose the admin and the redirector service on ports defined in .env file. It will start a separate postgres database on 5432 and redis server on 6379.
7. Access the application APIs:
   - Admin service: `http://localhost:{port}/api/v1/{users, urls}`
   - Redirector service: `http://localhost:{redirector_port}/{short_port}`
8. Test the application:
   - Create a user with the email and password
   - Login with the email and password and get the access token
   - Use the access token to access the APIs
