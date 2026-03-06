package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"cloud.google.com/go/spanner/apiv1"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/api/option"
)

func main() {
	project := flag.String("project", "", "GCP project ID (required)")
	instance := flag.String("instance", "", "Spanner instance ID (required)")
	database := flag.String("database", "", "Spanner database name (required)")
	databaseRole := flag.String("database-role", "", "Spanner FGAC database role (optional, if empty creates a normal IAM-based session)")
	flag.Parse()

	if *project == "" || *instance == "" || *database == "" {
		flag.Usage()
		log.Fatal("--project, --instance, --database are required")
	}

	ctx := context.Background()

	client, err := spanner.NewClient(ctx, option.WithQuotaProject(*project))
	if err != nil {
		log.Fatalf("failed to create spanner client: %v", err)
	}
	defer client.Close()

	req := &spannerpb.CreateSessionRequest{
		Database: fmt.Sprintf("projects/%s/instances/%s/databases/%s", *project, *instance, *database),
	}
	if *databaseRole != "" {
		req.Session = &spannerpb.Session{
			CreatorRole: *databaseRole,
		}
	}

	session, err := client.CreateSession(ctx, req)
	if err != nil {
		if *databaseRole != "" {
			log.Fatalf("failed to create session with FGAC role %q: %v", *databaseRole, err)
		}
		log.Fatalf("failed to create session: %v", err)
	}

	fmt.Print(session.Name)
}
