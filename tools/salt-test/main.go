package main

import (
	"encoding/base64"
	"fmt"
	"os"

	"gote/pkg/crypto"
)

func main() {
	// Test with the LOCAL salt from password hash file
	localSaltB64 := "nzEAqvBA/Gu1yTgI9vrEdIqt9tO9et/Ni2YJFMcdWrE="
	password := "cdmk4$Io"

	fmt.Println("🧪 Testing LOCAL salt from password hash file")

	salt, err := base64.StdEncoding.DecodeString(localSaltB64)
	if err != nil {
		fmt.Printf("❌ Could not decode salt: %v\n", err)
		os.Exit(1)
	}

	key := crypto.DeriveKey(password, salt)
	fmt.Printf("✅ Key derived with local salt\n")
	fmt.Printf("   Salt: %s\n", localSaltB64)
	fmt.Printf("   Key length: %d bytes\n", len(key))

	// Try to decrypt a note
	encryptedData := "Xfp8tLi6mU2D8HaXG/Iml8A7tEPouGGrXJ23JteRNN0BrSpztQ03DU4TT4a54dtYbDreB1xMK5T6F/VHVk7tioq3KScrwkC2T65LwOTSKPNNVA3yeBopuVCDc24uQI+k8kGZqw7TyDfamAhjYqLYYZWcAaZXRgP7VCZaye89clbUbFvMmZlULrrvvJ2xT2Xb8cj2bHC6sYi7C+yPNH9MElhunnDjjNZrzRFGI39mYtXYYiPaZv/CTegcDpRGHhkcf73XtNkPJbzdlB8h6yQbJBXMGqRTv3aWad/Drez/XS9ByFbR877sKmZlCAHqkGq4AEVo6R+edsyr61u10dQrECtyc0TE4lPYGxTC2q45CLi3jjgwu+9ed1dJ/18CBaC5KJH"

	content, err := crypto.Decrypt(encryptedData, key)
	if err != nil {
		fmt.Printf("❌ Decryption failed: %v\n", err)
	} else {
		fmt.Printf("✅ SUCCESS! Decryption worked with local salt!\n")
		fmt.Printf("   Content: %s\n", content)
	}
}
