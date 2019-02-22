package main
import (
	"fmt"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/sqs"
    "github.com/jinzhu/gorm"
    _ "github.com/jinzhu/gorm/dialects/sqlite"
)

var svc *sqs.SQS

type User struct {
    Id int
    Name string
}

func main() {
	fmt.Println("Start SQS")
	setupSQS()
	fmt.Println("Start DB")
	setupDB()
}

func setupSQS() {
	sess, err := session.NewSession(&aws.Config{
        Endpoint: aws.String("http://localhost:9324"), 
        Credentials: credentials.AnonymousCredentials, 
        Region: aws.String("us-west-2")},
	)
	if err != nil {
        fmt.Println("Error", err)
        return
	}

	svc := sqs.New(sess)

	// create Queue
	svc.CreateQueue(&sqs.CreateQueueInput{
        QueueName: aws.String("user"),
        Attributes: map[string]*string{
            "DelaySeconds":           aws.String("60"),
            "MessageRetentionPeriod": aws.String("86400"),
        },
    })

	// create data
	qURL := "http://localhost:9324/queue/user"
	svc.SendMessage(&sqs.SendMessageInput{
        DelaySeconds: aws.Int64(10),
        MessageBody: aws.String("1"),
        QueueUrl: &qURL,
	})
}

func setupDB() {
	db, err := gorm.Open("sqlite3", "user.db")
    if err != nil {
        panic("failed to connect database")
    }
    defer db.Close()

    // Migrate and Insert Data
    db.AutoMigrate(&User{})
    db.Create(&User{Id: 1, Name: "Taro Yamada"})
    db.Create(&User{Id: 2, Name: "Taro Tanaka"})
}