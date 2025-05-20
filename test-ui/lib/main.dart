import 'package:flutter/material.dart';
import 'dart:developer'; // For log()

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

  // Placeholder for username and password controllers if we were to handle input
  // final TextEditingController _usernameController = TextEditingController();
  // final TextEditingController _passwordController = TextEditingController();
  
  // TODO: Add methods for actual login, registration, message sending/receiving later

  @override
  void dispose() {
    _messageInputController.dispose();
    // _usernameController.dispose();
    // _passwordController.dispose();
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
    // TODO: Add TextEditingControllers for username and password if needed for real functionality
    // final _usernameController = TextEditingController();
    // final _passwordController = TextEditingController();

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
            const TextField(
              // controller: _usernameController,
              decoration: InputDecoration(
                labelText: 'Username',
                border: OutlineInputBorder(),
                hintText: 'Enter your username',
              ),
              keyboardType: TextInputType.text,
            ),
            const SizedBox(height: 12),
            const TextField(
              // controller: _passwordController,
              decoration: InputDecoration(
                labelText: 'Password',
                border: OutlineInputBorder(),
                hintText: 'Enter your password',
              ),
              obscureText: true,
            ),
            const SizedBox(height: 24),
            ElevatedButton(
              style: ElevatedButton.styleFrom(padding: const EdgeInsets.symmetric(vertical: 16)),
              onPressed: () {
                // TODO: Implement actual login logic
                // 1. Get username and password from controllers
                // 2. Call API: http.post(Uri.parse(loginUrl), body: {'username': username, 'password': password})
                // 3. If successful, store token and set _isAuthenticated = true
                // 4. If error, show error message
                log("Login button pressed (placeholder)"); // Use log from dart:developer
                setState(() {
                  _isAuthenticated = true; // Simulate successful login for now
                });
              },
              child: const Text('Login'),
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
          onPressed: () {
            setState(() {
              _isAuthenticated = false; // Simulate logout
              _messages.clear(); // Clear messages on logout
            });
            log("Logout button pressed");
          },
          child: const Text("Logout (placeholder)"),
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
}
