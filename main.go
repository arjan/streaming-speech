package main

import (
	"context"
	"fmt"
	"io"
	"os"

	speech "cloud.google.com/go/speech/apiv2"
	"cloud.google.com/go/speech/apiv2/speechpb"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

var scope = "https://www.googleapis.com/auth/cloud-platform"

var config = &speechpb.RecognitionConfig{
	DecodingConfig: &speechpb.RecognitionConfig_ExplicitDecodingConfig{
		ExplicitDecodingConfig: &speechpb.ExplicitDecodingConfig{
			Encoding:          speechpb.ExplicitDecodingConfig_LINEAR16,
			SampleRateHertz:   8000,
			AudioChannelCount: 1,
		},
	},
	Model:         "long",
	LanguageCodes: []string{"nl-NL"},
	Features: &speechpb.RecognitionFeatures{
		ProfanityFilter:            false,
		EnableAutomaticPunctuation: true,
	},
}

func check(e error, reason string) {
	if e != nil {
		fmt.Println("Failed:", reason)
		panic(e)
	}
}

func main() {
	ctx := context.Background()

	b, err := os.ReadFile("google_credentials.json")
	check(err, "Read JSON credentials")

	creds, err := google.CredentialsFromJSON(ctx, b, scope)
	check(err, "Construct Google credentials")

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	fmt.Println("Creds:", creds.ProjectID)

	client, err := speech.NewClient(ctx, option.WithCredentials(creds))
	check(err, "Create speech client")
	defer client.Close()

	stream, err := client.StreamingRecognize(ctx)
	check(err, "Create streaming recognize")

	err = stream.Send(&speechpb.StreamingRecognizeRequest{
		Recognizer: fmt.Sprintf("projects/%s/locations/global/recognizers/_", creds.ProjectID),
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config: config,
				StreamingFeatures: &speechpb.StreamingRecognitionFeatures{
					InterimResults:            true,
					EnableVoiceActivityEvents: true,
				},
			},
		},
	})
	check(err, "Initial Stream.send")

	file, err := os.Open("example.wav")
	check(err, "Read example.wav")
	defer file.Close()

	buf := make([]byte, 1024)

	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			break
		}

		fmt.Printf("Send %d bytes\n", n)

		err = stream.Send(&speechpb.StreamingRecognizeRequest{
			StreamingRequest: &speechpb.StreamingRecognizeRequest_Audio{
				Audio: buf[:n],
			},
		})
		check(err, "Send chunk")

	}
	fmt.Println("OK")

	for {
		resp, err := stream.Recv()
		check(err, "Speech receive")

		fmt.Printf("Resp: %v\n", resp)
	}
}
