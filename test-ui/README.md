# Test UI (Flutter)

This directory contains a basic Flutter application intended for testing and interacting with the Go backend server of the Keeper project.

## Prerequisites

-   [Flutter SDK](https://flutter.dev/docs/get-started/install) installed on your system.
-   The Go backend server should be running.

## Getting Started

1.  **Navigate to the UI directory:**
    ```bash
    cd test-ui
    ```

2.  **Get Flutter packages:**
    (This step is usually needed if `pubspec.yaml` has new dependencies or if you cloned the repo. The current basic structure might not strictly need it initially if only default packages are used, but it's good practice.)
    ```bash
    flutter pub get
    ```

3.  **Run the Flutter application:**
    ```bash
    flutter run
    ```
    This will typically run the app on a connected device, an emulator/simulator, or a web browser (if configured for web).

## API Configuration

The configuration for the backend API endpoints can be found in:
`lib/src/core/config/api_config.dart`

This file contains:
-   Base URLs for the HTTP and WebSocket servers.
-   Specific endpoint paths for registration, login, etc.
-   Helper comments and example snippets on how to:
    -   Make HTTP requests for registration and login.
    -   Establish a WebSocket connection, including how to pass the authentication token.

## Development Notes

-   This is a very basic scaffold. The UI for registration, login, message display, and message sending needs to be implemented.
-   You would typically use Flutter packages like:
    -   `http` for making HTTP requests.
    -   `web_socket_channel` for WebSocket communication.
    -   A state management solution (Provider, Riverpod, BLoC, GetX, etc.) for managing application state (like authentication tokens, messages).
-   The example code within `api_config.dart` provides hints on using these packages.

## Debugging & Viewing Logs

When running the Flutter web application (either locally with `flutter run -d chrome` or via the Docker container at `http://localhost:8081`):

-   **Browser Developer Console:** Application logs, including those added for debugging API calls (from `api_config.dart` and `main.dart`), will appear in your web browser's developer console.
    -   To open the console:
        -   In Chrome: Right-click on the page -> "Inspect" -> "Console" tab.
        -   In Firefox: Right-click on the page -> "Inspect Element" -> "Console" tab.
        -   Other browsers have similar tools.
-   Look for messages prefixed with `api_config:` or `_loginUser:` to trace the URL construction and API call details.
