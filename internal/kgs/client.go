package kgs

import (
	"context"
	"fmt"
	"time"

	kgspb "github.com/Nikhil-O1O5/url-shortener/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client kgspb.KeyGenerationServiceClient
}

func NewClient(addr string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to KGS at %s: %w", addr, err)
	}

	return &Client{
		conn:   conn,
		client: kgspb.NewKeyGenerationServiceClient(conn),
	}, nil
}

func (c *Client) GetKey(ctx context.Context) (string, error) {
	resp, err := c.client.GetKey(ctx, &kgspb.GetKeyRequest{})
	if err != nil {
		return "", fmt.Errorf("get key from KGS: %w", err)
	}
	return resp.Key, nil
}

func (c *Client) ReturnKey(ctx context.Context, key string) error {
	_, err := c.client.ReturnKey(ctx, &kgspb.ReturnKeyRequest{Key: key})
	if err != nil {
		return fmt.Errorf("return key to KGS: %w", err)
	}
	return nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
