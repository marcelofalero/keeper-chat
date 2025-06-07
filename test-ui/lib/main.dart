import 'package:flutter/material.dart';
import 'dart:developer'; // For log()
import 'package:http/http.dart' as http;
import 'dart:convert'; // For jsonEncode and jsonDecode
// Assuming 'test_ui' is the package name in pubspec.yaml
import 'package:test_ui/src/core/config/api_config.dart';

void main() {
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Keeper Test UI',
      theme: ThemeData(
        primarySwatch: Colors.blue,
        visualDensity: VisualDensity.adaptivePlatformDensity,
      ),
      home: const ChatPage(), // New home page
    );
  }
}

class ChatPage extends StatefulWidget {
  const ChatPage({super.key});

  @override
  State<ChatPage> createState() => _ChatPageState();
}

class _ChatPageState extends State<ChatPage> {
  bool _isAuthenticated = false; // Manages view state: true for chat, false for login
  final List<String> _messages = []; // Stores chat messages for display
  final TextEditingController _messageInputController = TextEditingController();

  // Username and password controllers
  final TextEditingController _usernameController = TextEditingController();
  final TextEditingController _passwordController = TextEditingController();

  // Authentication & State variables
  String? _jwtToken;
  String? _loginErrorMsg;
  bool _isLoading = false; // To show loading indicator

  // TODO: Add methods for actual login, registration, message sending/receiving later

  @override
  void dispose() {
    _messageInputController.dispose();
    _usernameController.dispose(); // Dispose username controller
    _passwordController.dispose(); // Dispose password controller
    super.dispose();
  }

  // The build method will be expanded in subsequent steps
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(_isAuthenticated ? "Chat Room" : "Login / Register - Keeper"),
      ),
      body: Padding(
        padding: const EdgeInsets.all(16.0),
        child: _isAuthenticated ? _buildChatUi() : _buildLoginRegisterUi(),
      ),
    );
  }

  Widget _buildLoginRegisterUi() {
    // _usernameController and _passwordController are now instance members of _ChatPageState

    return Center( // Center the content
      child: SingleChildScrollView( // Allow scrolling if content overflows
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          crossAxisAlignment: CrossAxisAlignment.stretch, // Make buttons stretch
          children: <Widget>[
            const Text(
              'Welcome to Keeper Chat!',
              style: TextStyle(fontSize: 24, fontWeight: FontWeight.bold),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 24),
            // TODO: Replace with actual TextFormField for validation if building a real form
            TextField( // Changed from const TextField
              controller: _usernameController, // Assign controller
              decoration: const InputDecoration( // Can be const
                labelText: 'Username',
                border: OutlineInputBorder(),
                hintText: 'Enter your username',
              ),
              keyboardType: TextInputType.text,
            ),
            const SizedBox(height: 12),
            TextField( // Changed from const TextField
              controller: _passwordController, // Assign controller
              decoration: const InputDecoration( // Can be const
                labelText: 'Password',
                border: OutlineInputBorder(),
                hintText: 'Enter your password',
              ),
              obscureText: true,
            ),
            const SizedBox(height: 24),
            if (_loginErrorMsg != null && _loginErrorMsg!.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(bottom: 16.0),
                child: Text(
                  _loginErrorMsg!,
                  style: const TextStyle(color: Colors.red, fontWeight: FontWeight.bold),
                  textAlign: TextAlign.center,
                ),
              ),
            ElevatedButton(
              style: ElevatedButton.styleFrom(padding: const EdgeInsets.symmetric(vertical: 16)),
              onPressed: _isLoading ? null : _loginUser, // Updated onPressed
              child: _isLoading 
                  ? const SizedBox(
                      height: 20, // Consistent height for the indicator
                      width: 20, 
                      child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2.0)
                    )
                  : const Text('Login'), // Updated child
            ),
            const SizedBox(height: 12),
            OutlinedButton(
              style: OutlinedButton.styleFrom(padding: const EdgeInsets.symmetric(vertical: 16)),
              onPressed: () {
                // TODO: Implement actual registration logic
                // 1. Get username and password from controllers
                // 2. Call API: http.post(Uri.parse(registerUrl), body: {'username': username, 'password': password})
                // 3. If successful, maybe auto-login or show success message
                // 4. If error, show error message
                log("Register button pressed (placeholder)");
              },
              child: const Text('Register'),
            ),
            const SizedBox(height: 24),
            const Text(
              '// TODO: Implement API calls using details from test-ui/lib/src/core/config/api_config.dart',
              style: TextStyle(color: Colors.grey),
              textAlign: TextAlign.center,
            ),
          ],
        ),
      ),
    );
  }

  // Placeholder for _buildChatUi - to be implemented in the next step
  Widget _buildChatUi() {
    // Ensure _messageInputController is available from _ChatPageState
    // final _messages list is also available from _ChatPageState

    return Column(
      children: <Widget>[
        if (_jwtToken != null) // Display token if available
          Padding(
            padding: const EdgeInsets.all(8.0),
            child: Text(
              'Token: $_jwtToken',
              style: const TextStyle(fontSize: 10, color: Colors.grey),
              overflow: TextOverflow.ellipsis, // Handle long tokens
            ),
          ),
        const Text(
          "// TODO: Use FlutterSecureStorage for production token handling.",
          style: TextStyle(fontSize: 10, color: Colors.orange),
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 8),
        const Text(
          "// TODO: Implement WebSocket connection here.",
          style: TextStyle(color: Colors.grey),
        ),
        const Text(
          "// TODO: Listen for incoming messages and add to _messages list.",
          style: TextStyle(color: Colors.grey),
        ),
        const SizedBox(height: 8),
        Expanded(
          child: ListView.builder(
            reverse: true, // To show latest messages at the bottom
            itemCount: _messages.length,
            itemBuilder: (context, index) {
              // Display messages in reverse order to simulate chat flow
              final message = _messages[_messages.length - 1 - index];
              // TODO: Differentiate between user's messages and others' messages for styling
              return Align(
                // alignment: message.isMine ? Alignment.centerRight : Alignment.centerLeft, // Example
                alignment: Alignment.centerLeft, // Simple alignment for now
                child: Card(
                  margin: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                  child: Padding(
                    padding: const EdgeInsets.all(10.0),
                    child: Text(message), // In a real app, this would be a Message object
                  ),
                ),
              );
            },
          ),
        ),
        Padding(
          padding: const EdgeInsets.all(8.0),
          child: Row(
            children: <Widget>[
              Expanded(
                child: TextField(
                  controller: _messageInputController,
                  decoration: const InputDecoration(
                    hintText: 'Enter your message...',
                    border: OutlineInputBorder(),
                  ),
                  onSubmitted: (_) => _sendMessage(), // Allow sending with enter key
                ),
              ),
              const SizedBox(width: 8),
              IconButton(
                icon: const Icon(Icons.send),
                onPressed: _sendMessage,
                tooltip: 'Send message',
              ),
            ],
          ),
        ),
        ElevatedButton(
          style: ElevatedButton.styleFrom(backgroundColor: Colors.orangeAccent), // Optional: Style logout button
          onPressed: () {
            setState(() {
              _isAuthenticated = false;
              _jwtToken = null; // Clear the token
              _messages.clear();
              _loginErrorMsg = null; // Also clear any lingering login errors
            });
            log("Logout button pressed. Token cleared.");
          },
          child: const Text("Logout"), // Removed "placeholder"
        ),
        const SizedBox(height: 8),
      ],
    );
  }

  void _sendMessage() {
    if (_messageInputController.text.trim().isEmpty) {
      return; // Don't send empty messages
    }
    final messageText = _messageInputController.text;
    // TODO: Send messageText via WebSocket
    // 1. Construct a message object/JSON
    // 2. channel.sink.add(jsonEncode(yourMessageObject));

    log("Send button pressed. Message: \"$messageText\" (placeholder)");

    setState(() {
      // For now, just add to local list to see it in UI
      // In a real app, server would echo back the message via WebSocket
      _messages.add("Me: $messageText");
      if (_messages.length > 10) { // Keep only last 10 messages for this placeholder
          _messages.removeAt(0);
      }
    });
    _messageInputController.clear();
  }

  Future<void> _loginUser() async {
    log('_loginUser: Method called. Username: "${_usernameController.text}", Password: "${_passwordController.text.isNotEmpty ? "provided" : "empty"}"');
    // This next log will also trigger the getter log from api_config.dart if it's the first access in a sequence
    log('_loginUser: Attempting to use loginUrl from config: $loginUrl');

    if (_usernameController.text.isEmpty || _passwordController.text.isEmpty) {
      setState(() {
        _loginErrorMsg = "Username and password cannot be empty.";
      });
      return;
    }

    setState(() {
      _isLoading = true;
      _loginErrorMsg = null; // Clear previous error
    });

    try {
      final uri = Uri.parse(loginUrl); // loginUrl is from api_config.dart
      log('_loginUser: Parsed URI: ${uri.toString()}');

      final requestBodyMap = <String, String>{
        'username': _usernameController.text,
        'password': _passwordController.text,
      };
      final requestBodyJson = jsonEncode(requestBodyMap);
      log('_loginUser: Request body (JSON): $requestBodyJson');

      final requestHeaders = {'Content-Type': 'application/json; charset=UTF-8'};
      log('_loginUser: Making POST request to ${uri.toString()} with headers: $requestHeaders');

      final response = await http.post(
        uri,
        headers: requestHeaders,
        body: requestBodyJson,
      );

      // log('Login response status: ${response.statusCode}'); // Existing log, can be kept or removed if too verbose
      // log('Login response body: ${response.body}'); // Existing log, can be kept or removed

      if (response.statusCode == 200) {
        final responseBody = jsonDecode(response.body);
        final token = responseBody['token'] as String?;
        if (token != null) {
          setState(() {
            _jwtToken = token;
            _isAuthenticated = true;
            _isLoading = false;
            _loginErrorMsg = null;
            // Clear text fields after successful login
            _usernameController.clear();
            _passwordController.clear();
          });
          log('Login successful. Token: $_jwtToken');
        } else {
          setState(() {
            _loginErrorMsg = 'Login successful, but no token received.';
            _isLoading = false;
          });
          log('Login successful but no token in response.');
        }
      } else {
        // Try to parse error from response body, e.g. { "error": "message" }
        String serverError = 'Invalid username or password.'; // Default
        try {
          final responseBody = jsonDecode(response.body);
          if (responseBody['error'] != null) {
            serverError = responseBody['error'] as String;
          }
        } catch (e) {
          // Ignore if response body is not JSON or doesn't have 'error'
          log('Could not parse error from response body: $e');
        }
        setState(() {
          _loginErrorMsg = 'Login failed: $serverError (Status: ${response.statusCode})';
          _isLoading = false;
        });
        log('Login failed. Status: ${response.statusCode}, Body: ${response.body}');
      }
    } catch (e) {
      log('_loginUser: Exception during HTTP POST: $e'); // Confirm this log exists
      setState(() {
        _loginErrorMsg = 'An error occurred during login: $e';
        _isLoading = false;
      });
    }
  }
}
