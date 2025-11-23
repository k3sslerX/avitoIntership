build:
	cd prReviewersAutoAssigner && go build -a -installsuffix cgo -o server ./cmd/server

build-image:
	cd prReviewersAutoAssigner && docker-compose build

run-container:
	cd prReviewersAutoAssigner && docker-compose up -d

load-test:
	cd prReviewersAutoAssigner && docker-compose up -d
	cd scripts && ./load_test.sh
	cd prReviewersAutoAssigner && docker-compose down
