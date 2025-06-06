// lib/src/core/config/api_config.dart
import 'dart:developer'; // For log()

// TODO: Consider using environment variables for baseUrl in a real app.
const String defaultBaseUrl = "localhost:8080"; 

// Attempt to get baseUrl from environment, fallback to default.
// Note: For Flutter web, --dart-define might be used. For mobile, other mechanisms.
// This simple string approach is for basic test UI purposes.
const String baseUrl = String.fromEnvironment('API_BASE_URL', defaultValue: defaultBaseUrl);

// Note: baseUrl's value is logged from within each getter when it's first accessed.

String get registerUrl {
  log('api_config: baseUrl value is: $baseUrl');
  final constructedUrl = "http://$baseUrl/api/register";
  log('api_config: registerUrl getter returning: $constructedUrl');
  return constructedUrl;
}

String get loginUrl {
  log('api_config: baseUrl value is: $baseUrl');
  final constructedUrl = "http://$baseUrl/api/login";
  log('api_config: loginUrl getter returning: $constructedUrl');
  return constructedUrl;
}

String get webSocketBaseUrl {
  log('api_config: baseUrl value is: $baseUrl');
  final constructedUrl = "ws://$baseUrl/ws";
  log('api_config: webSocketBaseUrl getter returning: $constructedUrl');
  return constructedUrl;
}

// Example of how to construct the WebSocket URL with a token:
// String getWebSocketUrlWithToken(String token) {
//   return "\$webSocketBaseUrl?token=\$token";
// }

/*
  How to use these constants:

  For HTTP requests (e.g., using the 'http' package):
  import 'package:http/http.dart' as http;
  import 'dart:convert'; // For jsonEncode/jsonDecode

  Future<void> loginUser(String username, String password) async {
    final response = await http.post(
      Uri.parse(loginUrl),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'username': username, 'password': password}),
    );
    if (response.statusCode == 200) {
      // Parse token from response.body
      // final token = jsonDecode(response.body)['token'];
      // print('Login successful, token: \$token');
    } else {
      // Handle error
      // print('Login failed: \${response.statusCode}');
    }
  }

  For WebSocket connections (e.g., using the 'web_socket_channel' package):
  import 'package:web_socket_channel/web_socket_channel.dart';

  void connectWebSocket(String token) {
    final wsUrl = getWebSocketUrlWithToken(token);
    // final channel = WebSocketChannel.connect(Uri.parse(wsUrl));
    
    // channel.stream.listen((message) {
    //   print('Received from WebSocket: \$message');
    // }, onError: (error) {
    //   print('WebSocket error: \$error');
    // }, onDone: () {
    //   print('WebSocket connection closed');
    // });

    // To send a message:
    // channel.sink.add('Hello from Flutter!');
  }
*/
