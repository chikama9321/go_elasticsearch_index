package main
import (
    "fmt"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/sqs"
    "github.com/jinzhu/gorm"
    _ "github.com/jinzhu/gorm/dialects/sqlite"
    "github.com/elastic/go-elasticsearch"
)

var svc *sqs.SQS

type User struct {
    Id int
    Name string
}

func main() {
    ids := dequeue()
    users := selectIndexData(ids)
    fmt.Println("users", users)

    es, _ := elasticsearch.NewDefaultClient()
    fmt.Println(es.Info())
}

/*
* dequeue index_target_ids fron Amazon SQS
*/
func dequeue() []string {
    sess, err := session.NewSession(&aws.Config{
        Endpoint: aws.String("http://localhost:9324"), 
        Credentials: credentials.AnonymousCredentials, 
        Region: aws.String("us-west-2")},
	)
	svc := sqs.New(sess)
    qURL := "http://localhost:9324/queue/user"

    result, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
        AttributeNames: []*string{
            aws.String(sqs.MessageSystemAttributeNameSentTimestamp),
        },
        MessageAttributeNames: []*string{
            aws.String(sqs.QueueAttributeNameAll),
        },
        QueueUrl:            &qURL,
        MaxNumberOfMessages: aws.Int64(3),
        VisibilityTimeout:   aws.Int64(20),  // 20 seconds
        WaitTimeSeconds:     aws.Int64(0),
    })

    if err != nil {
        fmt.Println("Error", err)
        return []string{}
    }

    if len(result.Messages) == 0 {
        fmt.Println("Received no messages")
        return []string{}
    }
    
    ids := []string{}
    for _, v := range result.Messages {
        ids = append(ids, *v.Body)
    }

    // for delete duplication key
    m := make(map[string]struct{})
    uniqueIds := make([]string, 0)

    for _, v := range ids {
        // mapでは、第二引数にその値が入っているかどうかの真偽値が入っている
        if _, ok := m[v]; !ok {
            m[v] = struct{}{}
            uniqueIds = append(uniqueIds, v)
        }
    }

    return uniqueIds
}

/*
* read data from db for index elasticsearch
*/
func selectIndexData(ids []string) []User {
    db, err := gorm.Open("sqlite3", "user.db")
    if err != nil {
        panic("failed to connect database")
    }
    defer db.Close()

    // Select
    users := make([]User, 0)
    db.Where("id in (?)", ids).Find(&users)
    return users
}