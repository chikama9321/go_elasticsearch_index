package main
import (
    "context"
    "strconv"
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
    Id int      `json:"id"`
    Name string `json:"name"`
}

func main() {
    ids := dequeue()
    users := selectIndexData(ids)
    indexES(users)
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
        panic(err)
    }

    if len(result.Messages) == 0 {
        panic(err)
    }
    
    ids := []string{}
    for _, v := range result.Messages {
        ids = append(ids, *v.Body)
    }

    // for delete duplication key
    m := make(map[string]struct{})
    uniqueIds := make([]string, 0)

    for _, v := range ids {
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

/*
* bulk index to elasticsearch
*/
func indexES(users []User) {
    ctx := context.Background()
	client, err := elastic.NewClient(
    	elastic.SetURL("http://localhost:9200"),
      	elastic.SetSniff(false),
   	)
   	if err != nil {
    	panic(err)
   	}
    defer client.Stop()
    
    
    bulkRequest := client.Bulk()
    for _, user := range users {
        idStr := strconv.Itoa(user.Id)
        index := elastic.NewBulkIndexRequest().Index("user").Type("user").Id(idStr).Doc(user)
        bulkRequest = bulkRequest.Add(index)
    }

    if _, err := bulkRequest.Do(ctx); err != nil {
        panic(err)
    }
}