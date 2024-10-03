package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type Request struct {
	Profile    string  `json:"profile"`
	Datastream string  `json:"datastream"`
	Tailoring  *string `json:"tailoring,omitempty"`
}

func handler(ctx context.Context, request Request) (events.APIGatewayProxyResponse, error) {
	customizations, err := processRequest(request.Profile, request.Datastream, request.Tailoring)
	if err != nil {
		response := events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       err.Error(),
		}
		return response, nil
	}

	response := events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(customizations),
	}
	return response, nil
}

func main() {
	lambda.Start(handler)
}
