A simple short URL service for a small group of my friends, implemented in Go and running on Google App Engine.

Note that actually running this locally requires creating a file called `secrets.go` in the `hms/` directory and adding a package level `map[string]<anything>` called `ALLOWED_EMAILS`, where the string keys are the emails allowed to add URLs to the service. To run locally, you just need to allow "test@example.com"
