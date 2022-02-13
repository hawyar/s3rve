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
		sess, err := newSession(&aws.Config{})

		if err != nil {
			// redirect to home page for now
			fmt.Println(err)
			return c.Redirect("/")
		}

		buckets, err := s3.New(sess).ListBuckets(nil)

		if err != nil {
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
			"Document":    "index",
			"Title":       "index",
			"Buckets":     b,
			"BucketCount": len(b),
			"Region":      "us-west-2",
			"UpdatedAt":   time.Now().Format("01/02/2006"),
		})
	})

	app.Get("/bucket/:name", func(c *fiber.Ctx) error {
		fmt.Println("bucket name:", c.Params("name"))
		bucket := c.Params("name")

		if bucket == "" {
			return c.Redirect("/")
		}

		prefix := c.Query("prefix")

		fmt.Println(prefix)

		fmt.Printf("requested bucket: %s \n", c.Params("name"))

		sess, err := newSession(&aws.Config{})

		if err != nil {
			fmt.Println(err)
			return c.Redirect("/")
		}

		s3svc := s3.New(sess)

		b, err := s3svc.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket: aws.String(c.Params("prefi")),
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

		var u []string

		for k := range obj {
			u = append(u, k)
		}

		//return c.Render("bucket", fiber.Map{
		//	"Document":   b.Name,
		//	"Title":      b.Name,
		//	"Bucket":     u,
		//	"ItemsCount": len(u),
		//	"Region":     "global",
		//	"UpdatedAt":  time.Now().Format("01/02/2006"),
		//})

		return c.SendString(fmt.Sprintf("%+v", u))
	})

	// catch other routes to /
	app.Use(func(c *fiber.Ctx) error {
		return c.Redirect("/")
	})

	log.Println("listening on", port)
	log.Fatal(app.Listen(":" + port))
}
