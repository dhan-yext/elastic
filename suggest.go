// Copyright 2012 Oliver Eilhard. All rights reserved.
// Use of this source code is governed by a MIT-license.
// See http://olivere.mit-license.org/license.txt for details.

package elastic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// SuggestService returns suggestions for text.
type SuggestService struct {
	client     *Client
	pretty     bool
	debug      bool
	routing    string
	preference string
	indices    []string
	types      []string
	suggesters []Suggester
}

func NewSuggestService(client *Client) *SuggestService {
	builder := &SuggestService{
		client:     client,
		indices:    make([]string, 0),
		types:      make([]string, 0),
		suggesters: make([]Suggester, 0),
	}
	return builder
}

func (s *SuggestService) Index(index string) *SuggestService {
	s.indices = append(s.indices, index)
	return s
}

func (s *SuggestService) Indices(indices ...string) *SuggestService {
	s.indices = append(s.indices, indices...)
	return s
}

func (s *SuggestService) Type(typ string) *SuggestService {
	s.types = append(s.types, typ)
	return s
}

func (s *SuggestService) Types(types ...string) *SuggestService {
	s.types = append(s.types, types...)
	return s
}

func (s *SuggestService) Pretty(pretty bool) *SuggestService {
	s.pretty = pretty
	return s
}

func (s *SuggestService) Debug(debug bool) *SuggestService {
	s.debug = debug
	return s
}

func (s *SuggestService) Routing(routing string) *SuggestService {
	s.routing = routing
	return s
}

func (s *SuggestService) Preference(preference string) *SuggestService {
	s.preference = preference
	return s
}

func (s *SuggestService) Suggester(suggester Suggester) *SuggestService {
	s.suggesters = append(s.suggesters, suggester)
	return s
}

func (s *SuggestService) Do() (SuggestResult, error) {
	// Build url
	urls := "/"

	// Indices part
	indexPart := make([]string, 0)
	for _, index := range s.indices {
		indexPart = append(indexPart, cleanPathString(index))
	}
	urls += strings.Join(indexPart, ",")

	// TODO Types part
	typesPart := make([]string, 0)
	for _, typ := range s.types {
		typesPart = append(typesPart, cleanPathString(typ))
	}
	urls += strings.Join(typesPart, ",")

	// Suggest
	urls += "/_suggest"

	// Parameters
	params := make(url.Values)
	if s.pretty {
		params.Set("pretty", fmt.Sprintf("%v", s.pretty))
	}
	if s.routing != "" {
		params.Set("routing", s.routing)
	}
	if s.preference != "" {
		params.Set("preference", s.preference)
	}
	if len(params) > 0 {
		urls += "?" + params.Encode()
	}

	// Set up a new request
	req, err := s.client.NewRequest("POST", urls)
	if err != nil {
		return nil, err
	}

	// Set body
	body := make(map[string]interface{})

	// Suggesters
	for _, s := range s.suggesters {
		body[s.Name()] = s.Source(false)
	}

	req.SetBodyJson(body)

	if s.debug {
		out, _ := httputil.DumpRequestOut((*http.Request)(req), true)
		fmt.Printf("%s\n", string(out))
	}

	// Get response
	res, err := s.client.c.Do((*http.Request)(req))
	if err != nil {
		return nil, err
	}
	if err := checkResponse(res); err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if s.debug {
		out, _ := httputil.DumpResponse(res, true)
		fmt.Printf("%s\n", string(out))
	}

	// There is a _shard object that cannot be deserialized.
	// So we use json.RawMessage instead.
	var suggestions map[string]*json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&suggestions); err != nil {
		return nil, err
	}

	ret := make(SuggestResult)
	for name, result := range suggestions {
		if name != "_shards" {
			var s []Suggestion
			if err := json.Unmarshal(*result, &s); err != nil {
				return nil, err
			}
			ret[name] = s
		}
	}

	return ret, nil
}

type SuggestResult map[string][]Suggestion

type Suggestion struct {
	Text    string             `json:"text"`
	Offset  int                `json:"offset"`
	Length  int                `json:"length"`
	Options []suggestionOption `json:"options"`
}

type suggestionOption struct {
	Text  string  `json:"text"`
	Score float32 `json:"score"`
	Freq  int     `json:"freq"`
}