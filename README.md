# bgg-picker

This is just a simple, first Go project to help pick a game from a collection for board game nights. It uses the Board Game Geek API to get a user's pre-existing collection and then filters it based on criteria.

The main purpose of this project was for learning:

- Basic Go
- Structuring an API with Go
- Deploying Go straight to Lambda with Serverless

## Instructions for use

For local development, you will need to specify a HTTP Port for it to run on. You can simply start the server with:

`HTTP_PORT=8000 go run .`

Afterwards, you can simply access the API from `localhost:8000`.

For Lambda development, you will need to have already setup AWS CLI with an IAM user with sufficient permissions for IaC deployment. If you have, then run `make deploy` and it will deploy straight to AWS and provide you a url to use.

The API has a single `GET` endpoint at `/`, it requires the following 3 parameters in the query string:

- `username`: A string for the BGG username (you may use "MorallyQuestionable" as an example)
- `playerCount`: A simple integer for number of players
- `playTime`: A string with a value of "short", "medium", or "long"

## TODO

- [ ] Create a proper directory/package structure
- [ ] Implement Tests
- [ ] Add "weight" as a factor (not included in BGG collections endpoint, would require making additional requests for each game). This would also be a good opportunity to play around with Go concurrency.
- [ ] Add caching system (especially due to extra requests for weight)

## Q&A

**If you like TDD so much, why aren't there any tests?**

Because the first step in learning how to test an API in a new language is sometimes learning how to build the API to begin with. Basically this is an example of a TDD "spike", where I've put together something quick in order to learn.

**Why Go?**

I wanted to learn something new and different, and Go is VERY different from PHP and Javascript.

**Why on Earth is it all in one file?**

Honestly, I'm still learning Go and the finer details of how packages are designed.
