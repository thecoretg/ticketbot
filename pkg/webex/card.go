package webex

type AdaptiveCard struct {
	Body []Element `json:"body"`
}

type Element interface {
	cardElement()
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

func (i InputChoiceSet) cardElement() {}
func (t TextBlock) cardElement()      {}

type ActionSet struct {
	ElementBase
	Actions []Action `json:"actions"`
}

type Action interface {
	cardAction()
}

func (a ActionSet) cardElement() {}

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
