package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html"
)

type Bucket struct {
	Name         string
	CreationDate time.Time
}

type BucketRequest struct {
	Prefix string `query:"prefix"`
}

func newSession() (*session.Session, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-2")},
	)

	if err != nil {
		return nil, err
	}

	return sess, nil
}

func allBuckets(s *session.Session) ([]*s3.Bucket, error) {
	s3svc := s3.New(s)

	buckets, err := s3svc.ListBuckets(nil)

	if err != nil {
		return nil, err
	}

	return buckets.Buckets, nil
}

func main() {

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
		Format: "[${ip}]:${port} ${status} - ${method} ${path}\n",
	}))

	app.Use(func(c *fiber.Ctx) error {
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Download-Options", "noopen")
		c.Set("Strict-Transport-Security", "max-age=5184000")
		c.Set("X-Frame-Options", "SAMEORIGIN")
		c.Set("X-DNS-Prefetch-Control", "off")

		// Go to next middleware:
		return c.Next()
	})

	// app.Use(cache.New(cache.Config{
	// 	Next: func(c *fiber.Ctx) bool {
	// 		return c.Query("refresh") == "true"
	// 	},
	// 	Expiration:   30 * time.Minute,
	// 	CacheControl: true,
	// }))

	// beware of BucketRegionError: on bucket and item fetch

	app.Get("/", func(c *fiber.Ctx) error {

		sess, err := newSession()

		if err != nil {
			// redirect to home page for now
			fmt.Println(err)
			return c.Redirect("/")
		}

		buckets, errb := s3.New(sess).ListBuckets(nil)

		if errb != nil {
			// redirect to home page for now
			fmt.Println(errb)
			return c.Redirect("/")
		}

		b := []Bucket{}

		for _, bucket := range buckets.Buckets {
			b = append(b, Bucket{
				Name:         *bucket.Name,
				CreationDate: *bucket.CreationDate,
			})
		}

		return c.Render("index", fiber.Map{
			"Document":    "S3i - Buckets",
			"Title":       "Buckets",
			"Buckets":     b,
			"BucketCount": len(b),
			"Region":      "us-west-2",
			"UpdatedAt":   time.Now().Format("01/02/2006"),
		})
	})

	app.Get("/bucket/:name", func(c *fiber.Ctx) error {

		if c.Params("name") == "" {
			fmt.Println("No bucket name")
			return c.Redirect("/")
		}

		fmt.Printf("requested bucket: %s \n", c.Params("name"))

		// print query params if any
		if c.Query("prefix") != "" {
			fmt.Printf("requested prefix: %s \n", c.Query("prefix"))
		}

		sess, err := newSession()

		if err != nil {
			// redirect to home page for now
			fmt.Println(err)
			return c.Redirect("/")
		}

		s3svc := s3.New(sess)

		b, err := s3svc.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket: aws.String(c.Params("name")),
		})

		if err != nil {
			fmt.Println(err)
		}

		obj := make(map[string]string)

		for _, e := range b.Contents {
			if obj[*e.Key] == "" {
				first := strings.Split(*e.Key, "/")[0]

				// remove the first element from e.Key
				obj[first] = "/" + strings.Join(strings.Split(*e.Key, "/")[1:], "/")

			}
		}

		u := []string{}

		for k := range obj {
			u = append(u, k)
		}

		return c.Render("bucket", fiber.Map{
			"Document":   b.Name,
			"Title":      b.Name,
			"Bucket":     u,
			"ItemsCount": len(u),
			"Region":     "us-east-2",
			"UpdatedAt":  time.Now().Format("01/02/2006"),
		})
	})

	log.Println("listening on", port)
	log.Fatal(app.Listen(":" + port))
}
