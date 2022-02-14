package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html"
	// "github.com/gofiber/fiber/v2/middleware/cache"
)

type Bucket struct {
	Name         string
	CreationDate time.Time
}

func newSession(config *aws.Config) (*session.Session, error) {

	if config.Region == nil {
		config.Region = aws.String("us-east-2")
	}

	sess, err := session.NewSession(config)

	if err != nil {
		return nil, err
	}

	return sess, nil
}

func main() {

	if os.Getenv("AWS_PROFILE") == "" {
		log.Fatal("No profile detected. Set your profile before starting the server: export AWS_PROFILE=<profile>")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	engine := html.New("./public/views", ".html")

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Static("/", "./public")

	app.Use(logger.New(logger.Config{
		Format: "${status} - ${method} ${path}\n",
	}))

	// app.Use(cache.New(cache.Config{
	// 	Next: func(c *fiber.Ctx) bool {
	// 		return c.Query("refresh") == "true"
	// 	},
	// 	Expiration:   30 * time.Minute,
	// 	CacheControl: true,
	// }))

	app.Get("/", func(c *fiber.Ctx) error {
		sess, err := newSession(&aws.Config{})

		if err != nil {
			fmt.Println(err)
			return c.Redirect("/")
		}

		buckets, err := s3.New(sess).ListBuckets(nil)

		if err != nil {
			return c.Redirect("/")
		}

		var b []Bucket

		for _, bucket := range buckets.Buckets {
			b = append(b, Bucket{
				Name:         *bucket.Name,
				CreationDate: *bucket.CreationDate,
			})
		}

		return c.Render("index", fiber.Map{
			"Document":  "S3 objects",
			"Title":     "Buckets",
			"Buckets":   b,
		})
	})

	app.Get("/bucket/:name", func(c *fiber.Ctx) error {
		bucket := c.Params("name")

		if bucket == "" {
			return c.Redirect("/")
		}

		sess, err := newSession(&aws.Config{})

		if err != nil {
			fmt.Println(err)
			return c.Redirect("/")
		}

		s3svc := s3.New(sess)

		fmt.Println(c.Query("prefix"))

		objects, err := s3svc.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket: aws.String(bucket),
		})

		if err != nil {
			fmt.Println(err)
			return c.Redirect("/")
		}

		if err != nil {
			fmt.Println(err)
		}

		obj := make(map[string]string)

		for _, e := range objects.Contents {
			if obj[*e.Key] == "" {
				first := strings.Split(*e.Key, "/")[0]

				obj[first] = "/" + strings.Join(strings.Split(*e.Key, "/")[1:], "/")
			}
		}

		var u []string

		for k := range obj {
			fmt.Println(k)
			u = append(u, k)
		}

		sort.Strings(u)

		return c.Render("bucket", fiber.Map{
			"Document":  "S3 Objects",
			"Title":     bucket,
			"Bucket":    bucket,
			"Objects":   u,
		})
	})

	app.Use(func(c *fiber.Ctx) error {
		return c.Redirect("/")
	})

	log.Println("listening on", port)
	log.Fatal(app.Listen(":" + port))
}
