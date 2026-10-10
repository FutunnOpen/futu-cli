package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/futunn/futu/internal/client"
	"github.com/futunn/futu/internal/output"
	"github.com/futunn/futu/pkg/symbol"
)

// quoteRESTSpec is one BFF quote API registered as a CLI command.
type quoteRESTSpec struct {
	ID         string
	Cmd        string
	Group      string
	Path       string
	Method     string
	Wrap       string
	Positional string
	Extra      string
	Params     []quoteRESTParam
}

type quoteRESTParam struct {
	Name     string
	Kind     string
	Required bool
	Default  string
	Source   string
	Enum     string
}

func init() {
	specs := loadQuoteRESTSpecs()
	registered := registerQuoteRESTAggregateCommands(specs)
	for i := range specs {
		if registered[specs[i].Cmd] || specs[i].Group == simGroupID {
			continue
		}
		rootCmd.AddCommand(newQuoteRESTCommand(&specs[i]))
	}
}

func loadQuoteRESTSpecs() []quoteRESTSpec {
	return quoteRESTSpecs
}

type quoteRESTAggregate struct {
	Cmd             string
	Use             string
	Short           string
	SelectorFlag    string
	SelectorDefault string
	Variants        []quoteRESTVariant
}

type quoteRESTVariant struct {
	Value string
	Cmd   string
}

var quoteRESTAggregates = []quoteRESTAggregate{
	{
		Cmd:             "broker",
		Use:             "broker <symbol>",
		Short:           "Get broker-tracking data",
		SelectorFlag:    "view",
		SelectorDefault: "current",
		Variants: []quoteRESTVariant{
			{Value: "current", Cmd: "top-ten-brokers"},
			{Value: "history", Cmd: "top-ten-brokers-history"},
		},
	},
	{
		Cmd:             "capital",
		Use:             "capital <symbol>",
		Short:           "Get capital-flow data",
		SelectorFlag:    "view",
		SelectorDefault: "intraday",
		Variants: []quoteRESTVariant{
			{Value: "intraday", Cmd: "capital-flow"},
			{Value: "history", Cmd: "capital-flow-history"},
			{Value: "distribution", Cmd: "capital-distribution"},
		},
	},
	{
		Cmd:             "company",
		Use:             "company <symbol> [leader]",
		Short:           "Get company profile, executives, and operating data",
		SelectorFlag:    "view",
		SelectorDefault: "profile",
		Variants: []quoteRESTVariant{
			{Value: "profile", Cmd: "company-profile"},
			{Value: "executives", Cmd: "company-executives"},
			{Value: "background", Cmd: "company-executive-background"},
			{Value: "efficiency", Cmd: "company-operational-efficiency"},
		},
	},
	{
		Cmd:             "corporate-actions",
		Use:             "corporate-actions <symbol>",
		Short:           "Get corporate-action records",
		SelectorFlag:    "type",
		SelectorDefault: "dividends",
		Variants: []quoteRESTVariant{
			{Value: "splits", Cmd: "corporate-actions-stock-splits"},
			{Value: "buybacks", Cmd: "corporate-actions-buybacks"},
			{Value: "dividends", Cmd: "corporate-actions-dividends"},
			{Value: "rehab", Cmd: "corporate-actions-rehab"},
		},
	},
	{
		Cmd:             "economic-calendar",
		Use:             "economic-calendar [keyword]",
		Short:           "Get or search economic-calendar data",
		SelectorFlag:    "view",
		SelectorDefault: "hot",
		Variants: []quoteRESTVariant{
			{Value: "hot", Cmd: "economic-calendar-hot"},
			{Value: "search", Cmd: "economic-calendar-search"},
		},
	},
	{
		Cmd:             "financials",
		Use:             "financials <symbol>",
		Short:           "Get financial statements, revenue, and earnings-price data",
		SelectorFlag:    "view",
		SelectorDefault: "statements",
		Variants: []quoteRESTVariant{
			{Value: "statements", Cmd: "financials-statements"},
			{Value: "revenue", Cmd: "financials-revenue-breakdown"},
			{Value: "earnings-history", Cmd: "financials-earnings-price-history"},
			{Value: "earnings-move", Cmd: "financials-earnings-price-move"},
		},
	},
	{
		Cmd:             "ipo-list",
		Use:             "ipo-list",
		Short:           "Get IPO lists by market",
		SelectorFlag:    "market",
		SelectorDefault: "HK",
		Variants: []quoteRESTVariant{
			{Value: "HK", Cmd: "ipo-list-hk"},
			{Value: "US", Cmd: "ipo-list-us"},
			{Value: "SG", Cmd: "ipo-list-sg"},
			{Value: "MY", Cmd: "ipo-list-my"},
			{Value: "CN", Cmd: "ipo-list-cn"},
		},
	},
	{
		Cmd:             "option-data",
		Use:             "option-data <symbol>",
		Short:           "Get option analytics and strategy data",
		SelectorFlag:    "view",
		SelectorDefault: "expiration",
		Variants: []quoteRESTVariant{
			{Value: "expiration", Cmd: "option-expiration-date"},
			{Value: "strategy", Cmd: "option-strategy"},
			{Value: "spread", Cmd: "option-strategy-spread"},
			{Value: "probability", Cmd: "option-exercise-probability"},
			{Value: "volatility", Cmd: "option-volatility"},
		},
	},
	{
		Cmd:             "plate",
		Use:             "plate [symbol]",
		Short:           "Get plate lists, plate stocks, or a symbol's owner plates",
		SelectorFlag:    "view",
		SelectorDefault: "list",
		Variants: []quoteRESTVariant{
			{Value: "list", Cmd: "plate-list"},
			{Value: "stocks", Cmd: "plate-stock"},
			{Value: "owner", Cmd: "owner-plate"},
		},
	},
	{
		Cmd:             "research",
		Use:             "research <symbol>",
		Short:           "Get research ratings and reports",
		SelectorFlag:    "type",
		SelectorDefault: "consensus",
		Variants: []quoteRESTVariant{
			{Value: "morningstar", Cmd: "research-morningstar-report"},
			{Value: "consensus", Cmd: "research-analyst-consensus"},
			{Value: "rating", Cmd: "research-rating-summary"},
		},
	},
	{
		Cmd:             "search",
		Use:             "search [keyword]",
		Short:           "Search news, community content, or stock feeds",
		SelectorFlag:    "type",
		SelectorDefault: "news",
		Variants: []quoteRESTVariant{
			{Value: "news", Cmd: "news-search"},
			{Value: "community", Cmd: "community-search"},
			{Value: "feed", Cmd: "stock-feed"},
		},
	},
	{
		Cmd:             "shareholders",
		Use:             "shareholders <symbol>",
		Short:           "Get shareholders and insider ownership data",
		SelectorFlag:    "view",
		SelectorDefault: "overview",
		Variants: []quoteRESTVariant{
			{Value: "overview", Cmd: "shareholders-overview"},
			{Value: "institutional", Cmd: "shareholders-institutional"},
			{Value: "changes", Cmd: "shareholders-holding-changes"},
			{Value: "holder-detail", Cmd: "shareholders-holder-detail"},
			{Value: "insider-holders", Cmd: "insider-holder-list"},
			{Value: "insider-trades", Cmd: "insider-trade-list"},
		},
	},
	{
		Cmd:             "short",
		Use:             "short <symbol>",
		Short:           "Get short-selling data",
		SelectorFlag:    "type",
		SelectorDefault: "volume",
		Variants: []quoteRESTVariant{
			{Value: "volume", Cmd: "daily-short-volume"},
			{Value: "interest", Cmd: "short-interest"},
		},
	},
	{
		Cmd:             "watchlist",
		Use:             "watchlist [symbol...]",
		Short:           "Get or modify user watchlists",
		SelectorFlag:    "action",
		SelectorDefault: "list",
		Variants: []quoteRESTVariant{
			{Value: "groups", Cmd: "user-security-group"},
			{Value: "list", Cmd: "user-security"},
			{Value: "add", Cmd: "modify-user-security"},
			{Value: "delete", Cmd: "modify-user-security"},
			{Value: "move-out", Cmd: "modify-user-security"},
		},
	},
	{
		Cmd:             "valuation",
		Use:             "valuation [symbol]",
		Short:           "Get valuation metrics and valuation stock lists",
		SelectorFlag:    "view",
		SelectorDefault: "detail",
		Variants: []quoteRESTVariant{
			{Value: "detail", Cmd: "valuation-detail"},
			{Value: "plate-stocks", Cmd: "valuation-plate-stock-list"},
			{Value: "index-stocks", Cmd: "valuation-index-component-stock-list"},
			{Value: "index-plates", Cmd: "valuation-index-stock-plate-list"},
		},
	},
}

func registerQuoteRESTAggregateCommands(specs []quoteRESTSpec) map[string]bool {
	specByCmd := make(map[string]*quoteRESTSpec, len(specs))
	for i := range specs {
		specByCmd[specs[i].Cmd] = &specs[i]
	}

	registered := map[string]bool{}
	for _, aggregate := range quoteRESTAggregates {
		cmd := newQuoteRESTAggregateCommand(aggregate, specByCmd)
		if cmd == nil {
			continue
		}
		rootCmd.AddCommand(cmd)
		for _, variant := range aggregate.Variants {
			registered[variant.Cmd] = true
		}
	}
	return registered
}

func newQuoteRESTAggregateCommand(aggregate quoteRESTAggregate, specByCmd map[string]*quoteRESTSpec) *cobra.Command {
	variantSpecs := make(map[string]*quoteRESTSpec, len(aggregate.Variants))
	for _, variant := range aggregate.Variants {
		spec := specByCmd[variant.Cmd]
		if spec == nil {
			continue
		}
		variantSpecs[variant.Value] = spec
	}
	if len(variantSpecs) == 0 {
		return nil
	}

	cmd := &cobra.Command{
		Use:     aggregate.Use,
		GroupID: quoteGroupID,
		Short:   aggregate.Short,
		Example: quoteRESTAggregateExample(aggregate),
		RunE: func(cmd *cobra.Command, args []string) error {
			spec, err := selectedQuoteRESTSpec(cmd, aggregate, variantSpecs)
			if err != nil {
				return err
			}
			if err := applyQuoteRESTAggregateDefaults(cmd, aggregate); err != nil {
				return err
			}
			return runQuoteREST(cmd, args, spec)
		},
	}
	cmd.Flags().String(aggregate.SelectorFlag, aggregate.SelectorDefault, quoteRESTAggregateSelectorUsage(aggregate))
	addQuoteRESTAggregateFlags(cmd, aggregate, variantSpecs)
	return cmd
}

func selectedQuoteRESTSpec(cmd *cobra.Command, aggregate quoteRESTAggregate, variants map[string]*quoteRESTSpec) (*quoteRESTSpec, error) {
	value, err := cmd.Flags().GetString(aggregate.SelectorFlag)
	if err != nil {
		return nil, err
	}
	value = normalizeQuoteRESTSelectorValue(value)
	for variant, spec := range variants {
		if normalizeQuoteRESTSelectorValue(variant) == value {
			return spec, nil
		}
	}
	return nil, fmt.Errorf("invalid --%s %q; valid values: %s", aggregate.SelectorFlag, value, quoteRESTAggregateSelectorValues(aggregate))
}

func applyQuoteRESTAggregateDefaults(cmd *cobra.Command, aggregate quoteRESTAggregate) error {
	if aggregate.Cmd != "watchlist" {
		return nil
	}
	action, err := cmd.Flags().GetString(aggregate.SelectorFlag)
	if err != nil {
		return err
	}
	switch normalizeQuoteRESTSelectorValue(action) {
	case "add":
		return setUnchangedFlag(cmd, "op", "ADD")
	case "delete":
		return setUnchangedFlag(cmd, "op", "DEL")
	case "move-out":
		return setUnchangedFlag(cmd, "op", "MOVE_OUT")
	default:
		return nil
	}
}

func setUnchangedFlag(cmd *cobra.Command, name string, value string) error {
	if cmd.Flags().Changed(name) {
		return nil
	}
	return cmd.Flags().Set(name, value)
}

func normalizeQuoteRESTSelectorValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func quoteRESTAggregateSelectorUsage(aggregate quoteRESTAggregate) string {
	return fmt.Sprintf("Select endpoint variant (default: %s). Values: %s", aggregate.SelectorDefault, quoteRESTAggregateSelectorValues(aggregate))
}

func quoteRESTAggregateSelectorValues(aggregate quoteRESTAggregate) string {
	values := make([]string, 0, len(aggregate.Variants))
	for _, variant := range aggregate.Variants {
		values = append(values, variant.Value)
	}
	return strings.Join(values, "/")
}

func quoteRESTAggregateExample(aggregate quoteRESTAggregate) string {
	switch aggregate.Cmd {
	case "ipo-list":
		return "  futu ipo-list\n  futu ipo-list --market US --format json"
	case "economic-calendar":
		return "  futu economic-calendar\n  futu economic-calendar CPI --view search"
	case "watchlist":
		return "  futu watchlist\n  futu watchlist HK.00700 --action add --group-name All"
	case "search":
		return "  futu search US.AAPL\n  futu search AAPL --type community"
	case "plate":
		return "  futu plate\n  futu plate HK.LIST1045 --view stocks"
	default:
		return fmt.Sprintf("  futu %s HK.00700\n  futu %s HK.00700 --%s %s --format json",
			aggregate.Cmd, aggregate.Cmd, aggregate.SelectorFlag, aggregate.Variants[len(aggregate.Variants)-1].Value)
	}
}

func addQuoteRESTAggregateFlags(cmd *cobra.Command, aggregate quoteRESTAggregate, variantSpecs map[string]*quoteRESTSpec) {
	params := map[string]quoteRESTParam{}
	kinds := map[string]map[string]bool{}
	for _, variant := range aggregate.Variants {
		spec := variantSpecs[variant.Value]
		if spec == nil {
			continue
		}
		for _, p := range spec.Params {
			if skipQuoteRESTFlag(spec, p) {
				continue
			}
			name := flagName(p.Name)
			if cmd.Flags().Lookup(name) != nil {
				continue
			}
			params[name] = p
			if kinds[name] == nil {
				kinds[name] = map[string]bool{}
			}
			kinds[name][p.Kind] = true
		}
		if spec.Extra == "leader_name" && cmd.Flags().Lookup("leader") == nil {
			cmd.Flags().String("leader", "", "Executive name (default: first executive returned by the company executives endpoint)")
		}
	}

	names := make([]string, 0, len(params))
	for name := range params {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		p := params[name]
		kind := p.Kind
		if len(kinds[name]) > 1 {
			kind = "string"
		}
		addQuoteRESTFlag(cmd, p, kind)
	}
}

func addQuoteRESTFlag(cmd *cobra.Command, p quoteRESTParam, kind string) {
	name := flagName(p.Name)
	usage := quoteRESTUsage(p)
	switch kind {
	case "int":
		cmd.Flags().Int(name, 0, usage)
	case "bool":
		cmd.Flags().Bool(name, false, usage)
	case "float":
		cmd.Flags().Float64(name, 0, usage)
	case "strings":
		cmd.Flags().StringSlice(name, nil, usage)
	default:
		cmd.Flags().String(name, "", usage)
	}
}

func newQuoteRESTCommand(spec *quoteRESTSpec) *cobra.Command {
	group := spec.Group
	if group == "" {
		group = quoteGroupID
	}
	cmd := &cobra.Command{
		Use:     quoteRESTUse(spec),
		GroupID: group,
		Short:   quoteRESTShort(spec),
		Long:    spec.Method + " " + spec.Path,
		Example: quoteRESTExample(spec),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQuoteREST(cmd, args, spec)
		},
	}
	for _, p := range spec.Params {
		if skipQuoteRESTFlag(spec, p) {
			continue
		}
		addQuoteRESTFlag(cmd, p, p.Kind)
	}
	if spec.Extra == "leader_name" {
		cmd.Flags().String("leader", "", "Executive name (default: first executive returned by the company executives endpoint)")
	}
	return cmd
}

func skipQuoteRESTFlag(spec *quoteRESTSpec, p quoteRESTParam) bool {
	if p.Source == "path" && p.Name == "symbol" {
		return true
	}
	if p.Name == "code_list" && spec.Positional == "codes" {
		return true
	}
	if p.Name == "symbol" && spec.Positional == "symbol" {
		return true
	}
	if p.Name == "leader_name" {
		return true
	}
	return false
}

func skipQuoteRESTApply(spec *quoteRESTSpec, p quoteRESTParam) bool {
	if skipQuoteRESTFlag(spec, p) {
		return true
	}
	if spec.Positional == "keyword" && (p.Name == spec.Extra || p.Name == "symbol" || p.Name == "keyword") {
		return true
	}
	if spec.Positional == "plate" && p.Name == spec.Extra {
		return true
	}
	return false
}

func quoteRESTUse(spec *quoteRESTSpec) string {
	switch spec.Positional {
	case "symbol":
		if spec.Extra == "leader_name" {
			return spec.Cmd + " <symbol> [leader]"
		}
		return spec.Cmd + " <symbol>"
	case "codes":
		return spec.Cmd + " <symbol...>"
	case "keyword":
		return spec.Cmd + " [keyword]"
	case "plate":
		return spec.Cmd + " [symbol]"
	default:
		return spec.Cmd
	}
}

func quoteRESTExample(spec *quoteRESTSpec) string {
	switch spec.Positional {
	case "symbol":
		return fmt.Sprintf("  futu %s HK.00700\n  futu %s US.AAPL --format json", spec.Cmd, spec.Cmd)
	case "codes":
		return fmt.Sprintf("  futu %s HK.00700\n  futu %s HK.00700 US.AAPL", spec.Cmd, spec.Cmd)
	case "keyword":
		if spec.Extra == "symbol" {
			return fmt.Sprintf("  futu %s\n  futu %s US.AAPL", spec.Cmd, spec.Cmd)
		}
		return fmt.Sprintf("  futu %s\n  futu %s FUTU", spec.Cmd, spec.Cmd)
	case "plate":
		return fmt.Sprintf("  futu %s\n  futu %s HK.LIST1045", spec.Cmd, spec.Cmd)
	default:
		return fmt.Sprintf("  futu %s\n  futu %s --format json", spec.Cmd, spec.Cmd)
	}
}

func quoteRESTUsage(p quoteRESTParam) string {
	parts := []string{quoteRESTUsageName(p.Name)}
	if isQuoteRESTSymbolParam(p.Name) {
		parts = append(parts, "in MARKET.CODE format")
	}
	if p.Default != "" {
		parts = append(parts, fmt.Sprintf("(default: %s)", p.Default))
	} else if !p.Required {
		parts = append(parts, "(optional, omitted by default)")
	}
	if p.Enum != "" {
		parts = append(parts, "values: "+p.Enum)
	}
	return strings.Join(parts, " ")
}

func quoteRESTShort(spec *quoteRESTSpec) string {
	if spec.Group == newsGroupID {
		return "Call a news REST endpoint"
	}
	if spec.Group == simGroupID {
		return "Call a simulated-trading REST endpoint"
	}
	return "Call a quote REST endpoint"
}

func briefText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	for _, mark := range []string{".", "\n", "Example:"} {
		if i := strings.Index(text, mark); i > 0 {
			text = strings.TrimSpace(text[:i])
		}
	}
	return strings.TrimRight(text, ",; ")
}

func runQuoteREST(cmd *cobra.Command, args []string, spec *quoteRESTSpec) error {
	pathParams := map[string]string{}
	query := url.Values{}
	body := map[string]any{}
	hasBody := spec.Method != "GET"

	switch spec.Positional {
	case "symbol":
		if len(args) < 1 {
			return fmt.Errorf("symbol is required, for example HK.00700")
		}
		code, err := parseOneFutuCode(args[0])
		if err != nil {
			return err
		}
		pathParams["symbol"] = code
		if spec.Method != "GET" {
			body["symbol"] = code
		}
		if spec.Extra == "leader_name" {
			leader := ""
			if len(args) > 1 {
				leader = args[1]
			}
			if flagLeader, _ := cmd.Flags().GetString("leader"); flagLeader != "" {
				leader = flagLeader
			}
			if leader == "" {
				var err error
				leader, err = firstExecutiveName(cmd, code)
				if err != nil {
					return err
				}
			}
			putRESTValue(query, body, hasBody, quoteRESTParam{Name: "leader_name", Kind: "string", Source: "query", Required: true}, leader)
		}
	case "codes":
		if len(args) < 1 {
			return fmt.Errorf("at least one symbol is required")
		}
		codes, err := parseFutuCodes(args)
		if err != nil {
			return err
		}
		body["code_list"] = codes
		hasBody = true
	case "keyword":
		keyword := "HK.00700"
		field := spec.Extra
		if field == "" {
			field = "symbol"
		}
		if field == "keyword" {
			keyword = "CPI"
		}
		if field == "news_ids" {
			keyword = ""
		}
		for _, p := range spec.Params {
			if p.Name == field && p.Default != "" {
				keyword = resolveDateToken(p.Default)
				break
			}
		}
		if fv, err := cmd.Flags().GetString(flagName(field)); err == nil && fv != "" {
			keyword = resolveDateToken(fv)
		}
		if len(args) > 0 {
			keyword = args[0]
		}
		if keyword != "" {
			if isQuoteRESTSymbolParam(field) {
				code, err := parseOneFutuCode(keyword)
				if err != nil {
					return err
				}
				keyword = code
			}
			putRESTValue(query, body, hasBody, quoteRESTParam{Name: field, Kind: "string", Source: "query", Required: true}, keyword)
		}
	case "plate":
		plate := "HK.LIST1045"
		field := spec.Extra
		if field == "" {
			field = "plate_code"
		}
		for _, p := range spec.Params {
			if p.Name == field && p.Default != "" {
				plate = p.Default
				break
			}
		}
		if fv, err := cmd.Flags().GetString(flagName(field)); err == nil && fv != "" {
			code, err := parseOneFutuCode(fv)
			if err != nil {
				return err
			}
			plate = code
		}
		if len(args) > 0 {
			code, err := parseOneFutuCode(args[0])
			if err != nil {
				return err
			}
			plate = code
		}
		putRESTValue(query, body, hasBody, quoteRESTParam{Name: field, Kind: "string", Source: "query", Required: true}, plate)
	}

	for _, p := range spec.Params {
		if skipQuoteRESTApply(spec, p) {
			continue
		}
		value, ok, err := readRESTFlag(cmd, p)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if p.Source == "path" {
			switch typed := value.(type) {
			case string:
				if typed != "" {
					pathParams[p.Name] = typed
				}
			default:
				pathParams[p.Name] = fmt.Sprint(typed)
			}
			continue
		}
		putRESTValue(query, body, hasBody || p.Source == "body", p, value)
	}

	for _, p := range spec.Params {
		if p.Source != "path" || p.Name == "symbol" {
			continue
		}
		if pathParams[p.Name] != "" {
			continue
		}
		if p.Name == "acc_id" {
			id, err := firstSimAccountID(cmd)
			if err != nil {
				return err
			}
			pathParams["acc_id"] = id
			continue
		}
		if p.Required {
			return fmt.Errorf("parameter %s is required", p.Name)
		}
	}

	if spec.ID == "news.detail" && query.Get("news_ids") == "" {
		id, err := firstNewsID(cmd)
		if err != nil {
			return err
		}
		query.Set("news_ids", id)
	}

	if spec.ID == "quote.warrant_screen" {
		applyWarrantPageDefault(cmd, body)
	}

	opts := client.CallOpts{PathParams: pathParams}
	if len(query) > 0 {
		opts.Params = query
	}
	if hasBody && len(body) > 0 {
		opts.Body = body
	}
	result, err := quoteSvc.Call(cmdContext(cmd), spec.ID, opts)
	if err != nil {
		return err
	}
	printBFFResult(result, func() {
		renderQuoteRESTTable(decodeJSONValue(result.Data), spec.Wrap)
	})
	return nil
}

func applyWarrantPageDefault(cmd *cobra.Command, body map[string]any) {
	if cmd.Flags().Changed(flagName("stock_owner")) || cmd.Flags().Changed("screen-groups") {
		return
	}
	marketType, _ := cmd.Flags().GetInt("market-type")
	if cmd.Flags().Changed("market-type") && marketType != 1 {
		return
	}
	body["stock_owner"] = "HK.00700"
}

func parseOneFutuCode(raw string) (string, error) {
	sym, err := symbol.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parsing symbol: %w", err)
	}
	return sym.FutuCode(), nil
}

func isQuoteRESTSymbolParam(name string) bool {
	switch name {
	case "symbol", "code_list", "stock_owner", "filter_security":
		return true
	default:
		return false
	}
}

func firstSimAccountID(cmd *cobra.Command) (string, error) {
	result, err := quoteSvc.Call(cmdContext(cmd), "sim_trade.account_list", client.CallOpts{})
	if err != nil {
		return "", fmt.Errorf("auto-fetching simulated account failed; pass --acc-id: %w", err)
	}
	data := decodeJSONValue(result.Data)
	items := extractObjectList(data, "accounts")
	if len(items) == 0 {
		items = extractObjectList(data, "")
	}
	for _, item := range items {
		if id := jsonString(item, "account_id"); id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("no simulated account found; pass --acc-id")
}

func firstNewsID(cmd *cobra.Command) (string, error) {
	query := url.Values{}
	query.Set("source_ids", "12")
	result, err := quoteSvc.Call(cmdContext(cmd), "news.source_latest", client.CallOpts{Params: query})
	if err != nil {
		return "", fmt.Errorf("auto-fetching news ID failed; pass news_ids: %w", err)
	}
	items := extractObjectList(decodeJSONValue(result.Data), "news_list")
	if len(items) == 0 {
		items = extractObjectList(decodeJSONValue(result.Data), "")
	}
	for _, item := range items {
		if id := jsonString(item, "news_id"); id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("news list is empty; pass news_ids, for example post:21670914")
}

func firstExecutiveName(cmd *cobra.Command, code string) (string, error) {
	result, err := quoteSvc.Call(cmdContext(cmd), "quote.company_executives", client.CallOpts{
		PathParams: map[string]string{"symbol": code},
	})
	if err != nil {
		return "", fmt.Errorf("auto-fetching executive name failed; pass it explicitly: %w", err)
	}
	items := extractObjectList(decodeJSONValue(result.Data), "executives")
	if len(items) == 0 {
		items = extractObjectList(decodeJSONValue(result.Data), "")
	}
	for _, item := range items {
		for _, key := range []string{"leader_name", "name", "sc_name"} {
			if name := jsonString(item, key); name != "" {
				return name, nil
			}
		}
	}
	return "", fmt.Errorf("no executive name found; pass it as a positional argument or with --leader")
}

func readRESTFlag(cmd *cobra.Command, p quoteRESTParam) (any, bool, error) {
	name := flagName(p.Name)
	changed := cmd.Flags().Changed(name)
	if !changed {
		if p.Default != "" {
			value, err := parseRESTFlagDefault(name, p)
			return value, true, err
		}
		if !p.Required {
			return nil, false, nil
		}
	}
	switch p.Kind {
	case "int":
		if isRESTStringFlag(cmd, name) {
			raw, err := cmd.Flags().GetString(name)
			if err != nil {
				return nil, false, err
			}
			value, err := strconv.Atoi(raw)
			if err != nil {
				return nil, false, fmt.Errorf("--%s requires integer: %w", name, err)
			}
			return value, true, nil
		}
		v, err := cmd.Flags().GetInt(name)
		if err != nil {
			return nil, false, err
		}
		if !changed && p.Default == "" && v == 0 && !p.Required {
			return nil, false, nil
		}
		return v, true, nil
	case "bool":
		if isRESTStringFlag(cmd, name) {
			raw, err := cmd.Flags().GetString(name)
			if err != nil {
				return nil, false, err
			}
			value, err := strconv.ParseBool(raw)
			if err != nil {
				return nil, false, fmt.Errorf("--%s requires boolean: %w", name, err)
			}
			return value, true, nil
		}
		v, err := cmd.Flags().GetBool(name)
		if err != nil {
			return nil, false, err
		}
		if !changed && p.Default == "" && !v && !p.Required {
			return nil, false, nil
		}
		return v, true, nil
	case "float":
		if isRESTStringFlag(cmd, name) {
			raw, err := cmd.Flags().GetString(name)
			if err != nil {
				return nil, false, err
			}
			value, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return nil, false, fmt.Errorf("--%s requires number: %w", name, err)
			}
			return value, true, nil
		}
		v, err := cmd.Flags().GetFloat64(name)
		if err != nil {
			return nil, false, err
		}
		if !changed && p.Default == "" && v == 0 && !p.Required {
			return nil, false, nil
		}
		return v, true, nil
	case "strings":
		if isRESTStringFlag(cmd, name) {
			raw, err := cmd.Flags().GetString(name)
			if err != nil {
				return nil, false, err
			}
			v := strings.Split(raw, ",")
			if len(v) == 0 {
				return nil, false, nil
			}
			if p.Name == "code_list" {
				codes, err := parseFutuCodes(v)
				if err != nil {
					return nil, false, err
				}
				return codes, true, nil
			}
			return v, true, nil
		}
		v, err := cmd.Flags().GetStringSlice(name)
		if err != nil {
			return nil, false, err
		}
		if len(v) == 0 {
			return nil, false, nil
		}
		if p.Name == "code_list" {
			codes, err := parseFutuCodes(v)
			if err != nil {
				return nil, false, err
			}
			return codes, true, nil
		}
		return v, true, nil
	default:
		v, err := cmd.Flags().GetString(name)
		if err != nil {
			return nil, false, err
		}
		return parseRESTFlagString(name, p, v)
	}
}

func parseRESTFlagDefault(name string, p quoteRESTParam) (any, error) {
	raw := resolveStaticDefault(p.Default)
	switch p.Kind {
	case "int":
		value, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("--%s default requires integer: %w", name, err)
		}
		return value, nil
	case "bool":
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("--%s default requires boolean: %w", name, err)
		}
		return value, nil
	case "float":
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("--%s default requires number: %w", name, err)
		}
		return value, nil
	case "strings":
		if raw == "" {
			return []string{}, nil
		}
		values := strings.Split(raw, ",")
		if p.Name == "code_list" {
			codes, err := parseFutuCodes(values)
			if err != nil {
				return nil, err
			}
			return codes, nil
		}
		return values, nil
	default:
		value, _, err := parseRESTFlagString(name, p, raw)
		return value, err
	}
}

func isRESTStringFlag(cmd *cobra.Command, name string) bool {
	flag := cmd.Flags().Lookup(name)
	return flag != nil && flag.Value.Type() == "string"
}

func parseRESTFlagString(name string, p quoteRESTParam, value string) (any, bool, error) {
	value = resolveDateToken(value)
	if value == "" {
		return nil, false, nil
	}
	if p.Kind == "json" {
		var parsed any
		if err := json.Unmarshal([]byte(value), &parsed); err != nil {
			return nil, false, fmt.Errorf("--%s requires JSON: %w", name, err)
		}
		return parsed, true, nil
	}
	if isQuoteRESTSymbolParam(p.Name) {
		code, err := parseOneFutuCode(value)
		if err != nil {
			return nil, false, err
		}
		return code, true, nil
	}
	return value, true, nil
}

func putRESTValue(query url.Values, body map[string]any, hasBody bool, p quoteRESTParam, value any) {
	source := p.Source
	if source == "" {
		if hasBody {
			source = "body"
		} else {
			source = "query"
		}
	}
	if source == "path" {
		return
	}
	if source == "body" {
		body[p.Name] = value
		return
	}
	switch typed := value.(type) {
	case string:
		query.Set(p.Name, typed)
	case int:
		query.Set(p.Name, strconv.Itoa(typed))
	case bool:
		query.Set(p.Name, strconv.FormatBool(typed))
	case float64:
		query.Set(p.Name, strconv.FormatFloat(typed, 'f', -1, 64))
	case []string:
		for _, item := range typed {
			query.Add(p.Name, item)
		}
	default:
		raw, _ := json.Marshal(typed)
		query.Set(p.Name, string(raw))
	}
}

func flagName(name string) string {
	switch name {
	case "stock_owner":
		return "symbol"
	case "filter_security":
		return "filter-symbol"
	default:
		return strings.ReplaceAll(name, "_", "-")
	}
}

func quoteRESTUsageName(name string) string {
	switch name {
	case "stock_owner":
		return "symbol"
	case "filter_security":
		return "filter-symbol"
	default:
		return name
	}
}

func atoiDefault(raw string) int {
	raw = resolveStaticDefault(raw)
	if raw == "" {
		return 0
	}
	n, _ := strconv.Atoi(raw)
	return n
}

func atofDefault(raw string) float64 {
	raw = resolveStaticDefault(raw)
	if raw == "" {
		return 0
	}
	n, _ := strconv.ParseFloat(raw, 64)
	return n
}

func resolveStaticDefault(raw string) string {
	return resolveDateToken(raw)
}

func resolveDateToken(raw string) string {
	now := time.Now()
	switch {
	case raw == "TODAY" || raw == "TODAY_YMD":
		return now.Format("2006-01-02")
	case raw == "TODAY_COMPACT":
		return now.Format("20060102")
	case strings.HasPrefix(raw, "TODAY+") && strings.HasSuffix(raw, "_YMD"):
		n := atoiDefault(strings.TrimSuffix(strings.TrimPrefix(raw, "TODAY+"), "_YMD"))
		return now.AddDate(0, 0, n).Format("2006-01-02")
	case strings.HasPrefix(raw, "TODAY+") && strings.HasSuffix(raw, "_COMPACT"):
		n := atoiDefault(strings.TrimSuffix(strings.TrimPrefix(raw, "TODAY+"), "_COMPACT"))
		return now.AddDate(0, 0, n).Format("20060102")
	case strings.HasPrefix(raw, "TODAY-") && strings.HasSuffix(raw, "_YMD"):
		n := atoiDefault(strings.TrimSuffix(strings.TrimPrefix(raw, "TODAY-"), "_YMD"))
		return now.AddDate(0, 0, -n).Format("2006-01-02")
	case strings.HasPrefix(raw, "TODAY-") && strings.HasSuffix(raw, "_COMPACT"):
		n := atoiDefault(strings.TrimSuffix(strings.TrimPrefix(raw, "TODAY-"), "_COMPACT"))
		return now.AddDate(0, 0, -n).Format("20060102")
	case strings.HasPrefix(raw, "TODAY+"):
		n := atoiDefault(strings.TrimPrefix(raw, "TODAY+"))
		return now.AddDate(0, 0, n).Format("2006-01-02")
	case strings.HasPrefix(raw, "TODAY-"):
		n := atoiDefault(strings.TrimPrefix(raw, "TODAY-"))
		return now.AddDate(0, 0, -n).Format("2006-01-02")
	default:
		return raw
	}
}

func renderQuoteRESTTable(data any, wrap string) {
	items := extractObjectList(data, wrap)
	if len(items) == 0 {
		if object, ok := data.(map[string]any); ok && object != nil {
			renderKeyValueTable(object)
			return
		}
		fmt.Fprintln(os.Stdout, "(empty)")
		return
	}
	headers, keys := pickTableColumns(items)
	tw := output.NewTableWriter(os.Stdout, headers...)
	for _, item := range items {
		row := make([]string, len(keys))
		for i, key := range keys {
			row[i] = jsonString(item, key)
		}
		tw.AddRow(row...)
	}
	tw.Render()
}

func extractObjectList(data any, wrap string) []map[string]any {
	if wrap != "" {
		if items := jsonObjectList(data, wrap); len(items) > 0 {
			return items
		}
	}
	if list, ok := data.([]any); ok {
		return anyListToObjects(list)
	}
	root, _ := data.(map[string]any)
	if root == nil {
		return nil
	}
	preferred := []string{wrap, "list", "items", "data"}
	for _, key := range preferred {
		if key == "" {
			continue
		}
		if items := jsonObjectList(data, key); len(items) > 0 {
			return items
		}
	}
	for _, value := range root {
		if list, ok := value.([]any); ok {
			items := anyListToObjects(list)
			if len(items) > 0 {
				return items
			}
		}
	}
	return nil
}

func pickTableColumns(items []map[string]any) ([]string, []string) {
	preferred := []string{
		"code", "symbol", "plate_code", "name", "sc_name", "plate_name",
		"title", "event_text", "leader_name", "time", "date", "last_price",
		"price", "change_rate", "volume", "market_state", "group_name", "group_type",
	}
	seen := map[string]bool{}
	keys := make([]string, 0, 8)
	for _, key := range preferred {
		if _, ok := items[0][key]; ok && !seen[key] {
			seen[key] = true
			keys = append(keys, key)
		}
	}
	if len(keys) < 4 {
		extra := make([]string, 0, len(items[0]))
		for key, value := range items[0] {
			if seen[key] {
				continue
			}
			switch value.(type) {
			case map[string]any, []any:
				continue
			}
			extra = append(extra, key)
		}
		sort.Strings(extra)
		for _, key := range extra {
			keys = append(keys, key)
			if len(keys) >= 8 {
				break
			}
		}
	}
	if len(keys) == 0 {
		keys = []string{"code"}
	}
	headers := make([]string, len(keys))
	for i, key := range keys {
		headers[i] = prettyHeader(key)
	}
	return headers, keys
}

func prettyHeader(key string) string {
	parts := strings.Split(key, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "")
}

func renderKeyValueTable(object map[string]any) {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	tw := output.NewTableWriter(os.Stdout, "Field", "Value")
	for _, key := range keys {
		value := object[key]
		switch typed := value.(type) {
		case map[string]any, []any:
			raw, _ := json.Marshal(typed)
			s := string(raw)
			if len(s) > 80 {
				s = s[:77] + "..."
			}
			tw.AddRow(key, s)
		default:
			tw.AddRow(key, jsonString(object, key))
		}
	}
	tw.Render()
}
