#!/bin/bash
# Generate self-signed certificates for local HTTPS testing

CERT_DIR="$(dirname "$0")/../certs"
mkdir -p "$CERT_DIR"

# Generate self-signed certificate
openssl req -x509 -newkey rsa:4096 \
  -keyout "$CERT_DIR/key.pem" \
  -out "$CERT_DIR/cert.pem" \
  -days 365 \
  -nodes \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1,IP:$(hostname -I | awk '{print $1}')"

echo ""
echo "Certificates generated in $CERT_DIR/"
echo ""
echo "To trust the certificate on your devices:"
echo "  1. Copy cert.pem to your phone"
echo "  2. Install it as a trusted certificate"
echo ""
echo "Or in Chrome, just click 'Advanced' -> 'Proceed to localhost'"
