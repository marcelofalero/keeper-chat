# Stage 1: Build the Flutter web application
FROM ghcr.io/fluttercommunity/flutter:stable AS builder
# Using ghcr.io/fluttercommunity/flutter:stable which is a community maintained image with Flutter SDK.
# Alternative: cirrusci/flutter:stable, but check for latest stable tags.

WORKDIR /app

# Copy pubspec.yaml and pubspec.lock files
COPY pubspec.* ./
# If pubspec.lock doesn't exist initially, this might cause an error.
# It's better if pubspec.lock is committed or `flutter pub get` is robust enough.
# For a fresh project, pubspec.lock might not be there.
# The worker created a pubspec.yaml, let's assume `flutter pub get` handles it.

# Get Flutter dependencies
RUN flutter pub get

# Copy the rest of the application source code
COPY . .

# Build the web application
# --release is implied by `build web` but being explicit is fine.
# --web-renderer html is often more compatible than canvaskit for simple cases or if canvaskit causes issues.
# Default is auto. Let's stick to default for now unless issues arise.
RUN flutter build web --release

# Stage 2: Serve the Flutter web application with Nginx
FROM nginx:alpine

# Copy the built web application from the builder stage to Nginx's webroot
COPY --from=builder /app/build/web /usr/share/nginx/html

# Copy a custom Nginx configuration if needed (optional for basic setup)
# For example, to handle SPA routing:
# COPY nginx.conf /etc/nginx/conf.d/default.conf
# If not providing a custom nginx.conf, the default serves index.html.

# Expose port 80 (Nginx default)
EXPOSE 80

# Command to start Nginx
CMD ["nginx", "-g", "daemon off;"]
