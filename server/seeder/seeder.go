package seeder

import (
	"fmt"
	// "io/ioutil" // Deprecated, using os.ReadFile instead
	"log"
	"os" // For os.ReadFile and os.Stat
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
	"keeper/server/core/ports"
	"keeper/server/models"
)

// YAMLUser defines the structure for users in the YAML fixture file.
type YAMLUser struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// YAMLMessage defines the structure for messages in the YAML fixture file.
type YAMLMessage struct {
	User      string `yaml:"user"`
	Text      string `yaml:"text"`
	Timestamp string `yaml:"timestamp,omitempty"` // omitempty for optional field
}

// UserFixtures is a wrapper struct to match the 'users' top-level key in users.yaml.
type UserFixtures struct {
	Users []YAMLUser `yaml:"users"`
}

// MessageFixtures is a wrapper struct to match the 'messages' top-level key in messages.yaml.
type MessageFixtures struct {
	Messages []YAMLMessage `yaml:"messages"`
}

// fileExists checks if a file exists at the given path and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil { // Handle other errors from os.Stat, like permission issues
		log.Printf("Error checking if file %s exists: %v", path, err)
		return false
	}
	return !info.IsDir()
}

// LoadFixtures loads user and message data from YAML files into the database.
func LoadFixtures(userRepo ports.UserRepository, messageRepo ports.MessageRepository, authService ports.AuthService, fixturesDir string) error {
	// --- Load Users ---
	usersFile := filepath.Join(fixturesDir, "users.yaml")
	if fileExists(usersFile) {
		log.Printf("Found users.yaml, attempting to load users from %s...", usersFile)
		yamlFile, err := os.ReadFile(usersFile)
		if err != nil {
			log.Printf("Error reading %s: %v. Skipping user fixtures.", usersFile, err)
		} else {
			var userFix UserFixtures
			err = yaml.Unmarshal(yamlFile, &userFix)
			if err != nil {
				log.Printf("Error unmarshalling users from %s: %v. Skipping user fixtures.", usersFile, err)
			} else {
				log.Printf("Successfully unmarshalled %d user(s) from users.yaml", len(userFix.Users))
				for _, u := range userFix.Users {
					existingUser, errRepo := userRepo.GetUserByUsername(u.Username)
					if errRepo != nil {
						log.Printf("Error checking if user '%s' exists: %v. Skipping.", u.Username, errRepo)
						continue
					}
					if existingUser == nil {
						hashedPassword, errHash := authService.HashPassword(u.Password)
						if errHash != nil {
							log.Printf("Error hashing password for user '%s': %v. Skipping.", u.Username, errHash)
							continue
						}
						newUser := models.User{Username: u.Username, PasswordHash: hashedPassword}
						errCreate := userRepo.CreateUser(newUser)
						if errCreate != nil {
							log.Printf("Error creating user '%s': %v.", u.Username, errCreate)
						} else {
							log.Printf("Successfully created user: %s", u.Username)
						}
					} else {
						log.Printf("User '%s' already exists. Skipping.", u.Username)
					}
				}
			}
		}
	} else {
		log.Printf("users.yaml not found in %s. No users loaded from fixtures.", fixturesDir)
	}

	// --- Load Messages ---
	messagesFile := filepath.Join(fixturesDir, "messages.yaml")
	if fileExists(messagesFile) {
		log.Printf("Found messages.yaml, attempting to load messages from %s...", messagesFile)
		yamlFile, err := os.ReadFile(messagesFile)
		if err != nil {
			log.Printf("Error reading %s: %v. Skipping message fixtures.", messagesFile, err)
		} else {
			var messageFix MessageFixtures
			err = yaml.Unmarshal(yamlFile, &messageFix)
			if err != nil {
				log.Printf("Error unmarshalling messages from %s: %v. Skipping message fixtures.", messagesFile, err)
			} else {
				log.Printf("Successfully unmarshalled %d message(s) from messages.yaml", len(messageFix.Messages))
				for _, m := range messageFix.Messages {
					var timestamp time.Time
					if m.Timestamp != "" {
						parsedTime, errParse := time.Parse(time.RFC3339, m.Timestamp)
						if errParse != nil {
							log.Printf("Error parsing timestamp for message from user '%s' ('%s'): %v. Using current time.", m.User, m.Timestamp, errParse)
							timestamp = time.Now().UTC()
						} else {
							timestamp = parsedTime.UTC()
						}
					} else {
						log.Printf("Message from user '%s' has no timestamp, using current time.", m.User)
						timestamp = time.Now().UTC()
					}
					newMessage := models.Message{User: m.User, Text: m.Text, Timestamp: timestamp}
					// SaveMessage expects models.Message, which includes ID. The ID will be auto-generated.
					_, errSave := messageRepo.SaveMessage(newMessage) // SaveMessage in SQLite adapter returns (models.Message, error)
					if errSave != nil {
						log.Printf("Error saving message from user '%s', text '%s': %v", m.User, m.Text, errSave)
					} else {
						log.Printf("Successfully saved message from user: %s, text: %s", m.User, m.Text)
					}
				}
			}
		}
	} else {
		log.Printf("messages.yaml not found in %s. No messages loaded from fixtures.", fixturesDir)
	}
	// This function currently doesn't return an error to halt startup,
	// as fixture loading is considered a best-effort enhancement.
	// If critical, error handling could be propagated up.
	return nil
}
