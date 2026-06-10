package fdorg

import "context"

// GetFinishedResults reuses the fixtures parser but requests only FINISHED.
func (c *Client) GetFinishedResults(ctx context.Context) ([]Match, error) {
	body, err := c.get(ctx, "/v4/competitions/WC/matches?status=FINISHED")
	if err != nil {
		return nil, err
	}
	return parseMatches(body)
}
