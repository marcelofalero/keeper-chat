import 'package:flutter/material.dart';
import 'dart:developer'; // For log()
import 'package:http/http.dart' as http;
import 'dart:convert'; // For jsonEncode and jsonDecode
import 'package:intl/intl.dart'; // For date formatting
import 'package:web_socket_channel/web_socket_channel.dart';
import 'package:web_socket_channel/status.dart' as status;

// Assuming 'test_ui' is the package name in pubspec.yaml
import 'package:test_ui/src/core/config/api_config.dart';
import 'package:test_ui/src/models/message.dart'; // Import the Message model

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
  final List<Message> _messages = []; // Stores chat messages for display
  final TextEditingController _messageInputController = TextEditingController();

  // Username and password controllers
  final TextEditingController _usernameController = TextEditingController();
  final TextEditingController _passwordController = TextEditingController();

  // Authentication & State variables
  String? _jwtToken;
  String? _currentUsername; // To store the logged-in user's name
  String? _loginErrorMsg;
  bool _isLoading = false; // To show loading indicator

  WebSocketChannel? _channel; // WebSocket channel

  // TODO: Add method for actual registration later

  @override
  void dispose() {
    _messageInputController.dispose();
    _usernameController.dispose();
    _passwordController.dispose();
    _channel?.sink.close(status.goingAway); // Close WebSocket channel
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
              return _buildMessageItem(message);
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
          style: ElevatedButton.styleFrom(backgroundColor: Colors.orangeAccent),
          onPressed: _logoutUser, // Call _logoutUser method
          child: const Text("Logout"),
        ),
        const SizedBox(height: 8),
      ],
    );
  }

  void _sendMessage() {
    if (_messageInputController.text.trim().isEmpty || _currentUsername == null) {
      return; // Don't send empty messages or if username is not set
    }
    final messageText = _messageInputController.text;

    final newMessage = Message(
      // id: null, // ID will be set by server, null for client-originated unsent messages
      user: _currentUsername!,
      text: messageText,
      timestamp: DateTime.now(),
      isMine: true, // Messages sent by the user are always "mine"
    );

    log("Attempting to send message: User='${_currentUsername!}', Text='${messageText}'");

    if (_channel != null) {
      final messageMap = {
        'type': 'sendMessage',
        'payload': {'text': messageText}
      };
      _channel!.sink.add(jsonEncode(messageMap));
      _messageInputController.clear();
      log('Message sent to WebSocket.');
    } else {
      log('Cannot send message: WebSocket channel is not active.');
      // Optionally show error to user
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Not connected. Cannot send message.')),
      );
    }
  }

  Widget _buildMessageItem(Message message) {
    final alignment = message.isMine ? CrossAxisAlignment.end : CrossAxisAlignment.start;
    final color = message.isMine ? Colors.blue[100] : Colors.grey[300];
    // final textAlign = message.isMine ? TextAlign.end : TextAlign.start; // Not needed if card aligns children
    final timeFormat = DateFormat('HH:mm');

    return Container(
      margin: const EdgeInsets.symmetric(vertical: 4.0, horizontal: 8.0),
      child: Column(
        crossAxisAlignment: alignment,
        children: <Widget>[
          Padding( // Add padding around the username
            padding: const EdgeInsets.only(left: 2.0, right: 2.0, bottom: 2.0), // Small padding
            child: Text(
              message.isMine ? 'Me' : message.user,
              style: const TextStyle(fontSize: 12.0, color: Colors.black54, fontWeight: FontWeight.w500),
            ),
          ),
          Card(
            color: color,
            elevation: 1.0, // Softer elevation
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.only(
                topLeft: const Radius.circular(12.0),
                topRight: const Radius.circular(12.0),
                bottomLeft: message.isMine ? const Radius.circular(12.0) : const Radius.circular(0), // Pointy corner for received
                bottomRight: message.isMine ? const Radius.circular(0) : const Radius.circular(12.0), // Pointy corner for sent
              )
            ),
            child: Padding(
              padding: const EdgeInsets.symmetric(vertical: 8.0, horizontal: 12.0),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start, // Text within card always starts left
                children: [
                  Text(
                    message.text,
                    style: const TextStyle(fontSize: 16.0, color: Colors.black87), // Slightly darker text
                  ),
                  const SizedBox(height: 4.0),
                  Text(
                    timeFormat.format(message.timestamp.toLocal()), // Display time in local timezone
                    style: TextStyle(fontSize: 10.0, color: Colors.grey[700]), // Darker grey for time
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
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

      // Create Basic Auth credentials
      final String credentials = '${_usernameController.text}:${_passwordController.text}';
      final String encodedCredentials = base64Encode(utf8.encode(credentials));
      log('_loginUser: Basic Auth encoded credentials: $encodedCredentials');

      final requestHeaders = {
        'Authorization': 'Basic $encodedCredentials',
      };
      log('_loginUser: Making POST request to ${uri.toString()} with Basic Auth header.');

      final response = await http.post(
        uri,
        headers: requestHeaders,
        // No body for Basic Auth token request as per current server setup
      );

      // log('Login response status: ${response.statusCode}');
      // log('Login response body: ${response.body}'); // Existing log, can be kept or removed

      if (response.statusCode == 200) {
        final responseBody = jsonDecode(response.body);
        final token = responseBody['token'] as String?;
        if (token != null) {
          setState(() {
            _jwtToken = token;
            _currentUsername = _usernameController.text; // Store username
            _isAuthenticated = true;
            _isLoading = false;
            _loginErrorMsg = null;
            _passwordController.clear(); // Definitely clear password field
          });
          log('Login successful. Token: $_jwtToken, User: $_currentUsername');
          _connectWebSocket(); // Connect to WebSocket
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

  void _connectWebSocket() {
    if (_jwtToken == null || _currentUsername == null) {
      log('Error: Cannot connect to WebSocket. Token or username is missing.');
      return;
    }

    // Ensure webSocketBaseUrl ends with /ws and doesn't have double slashes if apiBaseUrl already ends with /
    final wsBase = webSocketBaseUrl.endsWith('/') ? webSocketBaseUrl.substring(0, webSocketBaseUrl.length -1) : webSocketBaseUrl;
    final wsUrl = Uri.parse('$wsBase?token=$_jwtToken');
    log('Connecting to WebSocket: $wsUrl');

    _channel = WebSocketChannel.connect(wsUrl);

    _channel!.stream.listen(
      (message) {
        log('Received from WebSocket: $message');
        if (!mounted) return; // Check if the widget is still in the tree

        final decodedMessage = jsonDecode(message);
        final messageType = decodedMessage['type'] as String?;
        final payload = decodedMessage['payload'];

        if (_currentUsername == null) {
          log('Error: _currentUsername is null. Cannot process message.');
          return;
        }
        
        setState(() {
          if (messageType == 'newMessage') {
            final newMessage = Message.fromJson(payload as Map<String, dynamic>, _currentUsername!);
            _messages.insert(0, newMessage);
          } else if (messageType == 'history') {
            if (payload != null && payload['messages'] != null) {
              final historyMessages = (payload['messages'] as List)
                  .map((item) => Message.fromJson(item as Map<String, dynamic>, _currentUsername!))
                  .toList();
              _messages.insertAll(0, historyMessages);
               // Sort messages by timestamp after inserting history, oldest first in the list for display
              _messages.sort((a, b) => a.timestamp.compareTo(b.timestamp));
            } else {
               log('Received history message with null payload or messages.');
            }
          } else if (messageType == 'error') {
            if (payload != null && payload['message'] != null) {
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(content: Text('Server error: ${payload['message']}')),
              );
            } else {
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(content: Text('Received an unknown server error.')),
              );
            }
          }
        });
      },
      onError: (error) {
        log('WebSocket error: $error');
        if (!mounted) return;
        setState(() {
          _channel = null;
        });
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('WebSocket error: $error. Connection lost.')),
        );
      },
      onDone: () {
        log('WebSocket connection closed');
        if (!mounted) return;
        setState(() {
          _channel = null;
        });
        if (_isAuthenticated) { // Only show "connection closed" if user was authenticated (i.e., not on purpose logout)
            ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('WebSocket connection closed.')),
          );
        }
      },
      cancelOnError: true, // Important to prevent further processing on error
    );
    log('WebSocket listener set up.');
  }

  void _logoutUser() {
    log('Logging out user: $_currentUsername');
    _channel?.sink.close(status.goingAway);
    setState(() {
      _channel = null;
      _isAuthenticated = false;
      _jwtToken = null;
      _currentUsername = null;
      _messages.clear();
      _loginErrorMsg = null;
      _usernameController.clear(); // Clear username field on logout
      _passwordController.clear(); // Clear password field on logout
    });
    log('User logged out. UI state reset.');
  }
}
