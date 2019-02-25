package main
import (
	"fmt"
	"context"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/sqs"
    "github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/olivere/elastic"
)

var svc *sqs.SQS

type User struct {
    Id int
    Name string
}

const mapping = `
{
	"settings":{
		"number_of_shards": 1,
		"number_of_replicas": 0
	},
	"mappings":{
		"user":{
			"properties":{
				"id":{
					"type":"keyword"
				},
				"name":{
					"type":"text"
				}
			}
		}
	}
}`

func main() {
	fmt.Println("Start SQS")
	setupSQS()
	fmt.Println("Start DB")
	setupDB()
	fmt.Println("Start ES")
	setupES()
}

func setupSQS() {
	sess, err := session.NewSession(&aws.Config{
        Endpoint: aws.String("http://elasticmq:9324"), 
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
	qURL := "http://elasticmq:9324/queue/user"
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

func setupES() {
	ctx := context.Background()
	client, err := elastic.NewClient(
    	elastic.SetURL("http://elasticsearch:9200"),
      	elastic.SetSniff(false),
   	)
   	if err != nil {
    	panic(err)
   	}
   	defer client.Stop()
	
	exists, err := client.IndexExists("user").Do(ctx)
	if err != nil {
		panic(err)
	}
	
	if exists {
		if _, err := client.DeleteIndex("user").Do(ctx); err != nil {
			panic(err)
		}
	}
	
	if _, err := client.CreateIndex("user").BodyString(mapping).Do(ctx); err != nil {
		panic(err)
	}
}