package main

// import (
// 	"fmt"
// 	"log"
// 	"os"

// 	"github.com/joho/godotenv"
// 	"github.com/mailjet/mailjet-apiv3-go/v4"
// )

// func main() {

// 	envErr := godotenv.Load()

// 	if envErr != nil {
// 		log.Fatal("Failed to load .env")
// 	}

// 	err := send()
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 	}
// }

// func send() error {
// 	mj := mailjet.NewMailjetClient(
// 		os.Getenv("MAILJET_KEY"),
// 		os.Getenv("MAILJET_SECRET"),
// 	)

// 	message := mailjet.InfoMessagesV31{
// 		From: &mailjet.RecipientV31{
// 			Email: os.Getenv("MAILJET_EMAIL"),
// 			Name:  os.Getenv("MAILJET_FROM"),
// 		},
// 		To: &mailjet.RecipientsV31{
// 			{
// 				Email: "kigongovincent625@gmail.com",
// 				Name:  "vincent",
// 			},
// 		},
// 		Subject:  "Test Email raaaa",
// 		TextPart: "Hello, this is a test email sent from Go using Mailjet.",
// 	}

// 	messages := mailjet.MessagesV31{
// 		Info: []mailjet.InfoMessagesV31{message},
// 	}

// 	// Send the email
// 	_, err := mj.SendMailV31(&messages)
// 	if err != nil {
// 		return err
// 	}

// 	fmt.Println("Email sent successfully!")
// 	return nil
// }
