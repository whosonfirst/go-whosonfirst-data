package resolver

/*

$> cd /usr/local/data/dynamodb
$> java -Djava.library.path=./DynamoDBLocal_lib -jar DynamoDBLocal.jar -sharedDb

$> cd /usr/local/whosonfirst/go-whosonfirst-findingaid
$> go run -mod vendor cmd/create-dynamodb-tables/main.go -dynamodb-uri 'awsdynamodb://findingaid?region=us-west-2&endpoint=http://localhost:8000&credentials=static:local:local:local' -refresh
$> make cli && ./bin/populate -producer-uri 'awsdynamodb://findingaid?region=us-west-2&endpoint=http://localhost:8000&credentials=static:local:local:local&partition_key=id' /usr/local/data/sfomuseum-data-maps/

$> cd /usr/local/whosonfirst/go-reader-findingaid
$> make cli && ./bin/read -reader-uri 'findingaid://awsdynamodb/findingaid?region=us-west-2&endpoint=http://localhost:8000&credentials=static:local:local:local&partition_key=id&template=https://raw.githubusercontent.com/sfomuseum-data/{repo}/main/data/' 1360391327 | jq '.["properties"]["wof:name"]'

"SFO (1988)"

*/

import (
	"context"
	"fmt"
	_ "log"
	"net/url"
	"strings"

	"github.com/aaronland/go-aws-dynamodb"
	"gocloud.dev/docstore"
	gc_dynamodb "gocloud.dev/docstore/awsdynamodb"
)

// type DocstoreResolver implements the `Resolver` interface for data stored in a gocloud.dev/docstore compatible collection.
type DocstoreResolver struct {
	Resolver
	// A Docstore `sql.DB` instance containing Who's On First finding aid data.
	collection *docstore.Collection
}

func init() {

	ctx := context.Background()

	RegisterResolver(ctx, "awsdynamodb", NewDocstoreResolver)

	for _, scheme := range docstore.DefaultURLMux().CollectionSchemes() {

		err := RegisterResolver(ctx, scheme, NewDocstoreResolver)

		if err != nil {
			panic(err)
		}
	}
}

// NewDocstoreResolver will return a new `Resolver` instance for resolving repository names
// and IDs stored in a gocloud.dev/docstore Collection.
func NewDocstoreResolver(ctx context.Context, uri string) (Resolver, error) {

	u, err := url.Parse(uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to parse URL, %w", err)
	}

	// START OF put me in a package function or something

	var collection *docstore.Collection

	if u.Scheme == "awsdynamodb" {

		// Connect local dynamodb using Golang
		// https://gist.github.com/Tamal/02776c3e2db7eec73c001225ff52e827
		// https://gocloud.dev/howto/docstore/#dynamodb-ctor

		client, err := dynamodb.NewClientWithURI(ctx, uri)

		if err != nil {
			return nil, fmt.Errorf("Failed to create client, %v", err)
		}

		u, _ := url.Parse(uri)
		table_name := u.Host

		table_name = strings.TrimLeft(table_name, "/")

		q := u.Query()

		partition_key := q.Get("partition_key")

		col, err := gc_dynamodb.OpenCollection(client, table_name, partition_key, "", nil)

		if err != nil {
			return nil, fmt.Errorf("Failed to open collection, %w", err)
		}

		collection = col

	} else {

		col, err := docstore.OpenCollection(ctx, uri)

		if err != nil {
			return nil, fmt.Errorf("Failed to create database for '%s', %w", uri, err)
		}

		collection = col
	}

	// END OF put me in a package function or something

	if err != nil {
		return nil, fmt.Errorf("Failed to open collection, %w", err)
	}

	f := &DocstoreResolver{
		collection: collection,
	}

	return f, nil
}

// GetRepo returns the name of the repository associated with this ID in a Who's On First finding aid.
func (r *DocstoreResolver) GetRepo(ctx context.Context, id int64) (string, error) {

	// TBD: Import whosonfirst/go-whosonfirst-findingaid/producer/docstore CatalogRecord?

	doc := map[string]interface{}{
		"id":        id,
		"repo_name": "",
	}

	err := r.collection.Get(ctx, doc)

	if err != nil {
		return "", fmt.Errorf("Failed to get record for %d, %w", id, err)
	}

	repo := doc["repo_name"].(string)
	return repo, nil
}
