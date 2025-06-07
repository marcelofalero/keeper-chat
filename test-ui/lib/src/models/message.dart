class Message {
  final int? id;
  final String user;
  final String text;
  final DateTime timestamp;
  final bool isMine;

  Message({
    this.id,
    required this.user,
    required this.text,
    required this.timestamp,
    required this.isMine,
  });

  factory Message.fromJson(Map<String, dynamic> json, String currentUsername) {
    return Message(
      id: json['ID'] as int?, // Server sends 'ID'
      user: json['User'] as String, // Server sends 'User'
      text: json['Text'] as String, // Server sends 'Text'
      timestamp: DateTime.parse(json['Timestamp'] as String), // Server sends 'Timestamp'
      isMine: json['User'] == currentUsername,
    );
  }
}
