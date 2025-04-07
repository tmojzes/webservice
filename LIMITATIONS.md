# Limitations about the current design

## Timestamp Handling

* The timestamp is currently stored only in memory, so it is lost when the application restarts.
* Persistent storage (e.g., file, database) is recommended.

## Code Structure

* The code structure should be better separated for improved readability and testability.

## Monitoring and Logging

* At least one logging middleware that logs requests is missing.
* A Prometheus middleware for metrics is missing.
* Logs should be forwarded to a central log collector (Loki, Elastic, etc.).

## Health Check

* An endpoint for health checks for Kubernetes should be added.

## Development Environment

* Basic components like Dockerfile, Makefile, linter and CI configurations are missing.
* It is recommended to add configurations necessary for setting up the development environment.

## Version Control

* Currently, it's not implemented, but the application version should be built into the application.
* This can be essential for CLI applications.
