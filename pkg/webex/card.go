package webex

type AdaptiveCard struct {
	ContentType string              `json:"contentType"`
	Content     AdaptiveCardContent `json:"content"`
}

const (
	cardContentType = "application/vnd.microsoft.card.adaptive"
	cardSchema      = "http://adaptivecards.io/schemas/adaptive-card.json"
	cardVersion     = "1.3"
	cardType        = "AdaptiveCard"
)

type AdaptiveCardContent struct {
	Schema  string    `json:"$schema"`
	Type    string    `json:"type"`
	Version string    `json:"version"`
	Body    []Element `json:"body"`
}

func NewAdaptiveCard() *AdaptiveCard {
	return &AdaptiveCard{
		ContentType: cardContentType,
		Content: AdaptiveCardContent{
			Schema:  cardSchema,
			Type:    cardType,
			Version: cardVersion,
			Body:    []Element{},
		},
	}
}

func (c *AdaptiveCard) AddTextBlock(t *TextBlock) *AdaptiveCard {
	t.Type = t.typeString()
	c.Content.Body = append(c.Content.Body, t)
	return c
}

func (c *AdaptiveCard) AddInputChoiceSet(i *InputChoiceSet) *AdaptiveCard {
	i.Type = i.typeString()
	c.Content.Body = append(c.Content.Body, i)
	return c
}

func (c *AdaptiveCard) AddActionSet(a *ActionSet) *AdaptiveCard {
	a.Type = a.typeString()
	c.Content.Body = append(c.Content.Body, a)
	return c
}

type Element interface {
	cardElement()
	typeString() string
}

type (
	Color          string
	FontSize       string
	FontType       string
	FontWeight     string
	HAlignment     string
	TextBlockStyle string
	BlockHeight    string
	Spacing        string
)

const (
	BlockHeightAuto    BlockHeight = "auto"
	BlockHeightStretch BlockHeight = "stretch"
	ColorDefault       Color       = "default"
	ColorDark          Color       = "dark"
	ColorLight         Color       = "light"
	ColorAccent        Color       = "accent"
	ColorGood          Color       = "good"
	ColorWarning       Color       = "warning"
	ColorAttention     Color       = "attention"

	FontSizeDefault    FontSize   = "default"
	FontSizeSmall      FontSize   = "small"
	FontSizeMedium     FontSize   = "medium"
	FontSizeLarge      FontSize   = "large"
	FontSizeExtraLarge FontSize   = "extraLarge"
	FontTypeDefault    FontType   = "default"
	FontTypeMonospace  FontType   = "monospace"
	FontWeightDefault  FontWeight = "default"
	FontWeightLighter  FontWeight = "lighter"
	FontWeightBolder   FontWeight = "bolder"

	HAlignLeft   HAlignment = "left"
	HAlignCenter HAlignment = "center"
	HAlignRight  HAlignment = "right"

	TextBlockStyleDefault TextBlockStyle = "default"
	TextBlockStyleHeading TextBlockStyle = "heading"
	SpacingDefault        Spacing        = "default"
	SpacingNone           Spacing        = "none"
	SpacingSmall          Spacing        = "small"
	SpacingMedium         Spacing        = "medium"
	SpacingLarge          Spacing        = "large"
	SpacingExtraLarge     Spacing        = "extraLarge"
	SpacingPadding        Spacing        = "padding"
)

type ElementBase struct {
	Type string `json:"type"`
}

type TextBlock struct {
	ElementBase
	Text                string     `json:"text,omitempty"`
	Color               Color      `json:"color,omitempty"`
	FontType            FontType   `json:"fontType,omitempty"`
	HorizontalAlignment HAlignment `json:"horizontalAlignment,omitempty"`
	IsSubtle            bool       `json:"isSubtle,omitempty"`
	MaxLines            int        `json:"maxLines,omitempty"`
	FontSize            FontSize   `json:"size,omitempty"`
	FontWeight          FontWeight `json:"weight,omitempty"`
	Wrap                bool       `json:"wrap,omitempty"`
}

func (t TextBlock) cardElement()       {}
func (t TextBlock) typeString() string { return "TextBlock" }

type InputChoiceSet struct {
	ElementBase
	ID            string        `json:"id"`
	Choices       []InputChoice `json:"choices"`
	IsMultiSelect bool          `json:"isMultiSelect,omitempty"`
}

type InputChoice struct {
	Title string `json:"title"`
	Value string `json:"value"`
}

func (i InputChoiceSet) cardElement()       {}
func (i InputChoiceSet) typeString() string { return "Input.ChoiceSet" }

type ActionSet struct {
	ElementBase
	Actions []Action `json:"actions"`
}

type Action interface {
	cardAction()
}

func (a ActionSet) cardElement()       {}
func (a ActionSet) typeString() string { return "ActionSet" }

type ActionOpenURL struct {
	ElementBase
	URL string `json:"url"`
}

func (a ActionOpenURL) cardAction() {}

type ActionSubmit struct {
	ElementBase
	Data map[string]any `json:"data,omitempty"`
}

func (a ActionSubmit) cardAction() {}
