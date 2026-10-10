package cmd

import (
	_ "embed"
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

//go:embed quote_rest_specs.json
var quoteRESTSpecsJSON []byte

// quoteRESTSpec is one BFF quote API registered as a CLI command.
type quoteRESTSpec struct {
	ID         string           `json:"id"`
	Cmd        string           `json:"cmd"`
	Group      string           `json:"group"`
	Path       string           `json:"path"`
	Method     string           `json:"method"`
	Wrap       string           `json:"wrap"`
	Short      string           `json:"short"`
	Positional string           `json:"positional"`
	Extra      string           `json:"extra"`
	Keyword    bool             `json:"keyword"`
	Params     []quoteRESTParam `json:"params"`
}

type quoteRESTParam struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Kind     string `json:"kind"`
	Required bool   `json:"required"`
	Default  string `json:"default"`
	Source   string `json:"source"`
	Desc     string `json:"desc"`
	Enum     string `json:"enum"`
	Pattern  string `json:"pattern"`
	Format   string `json:"format"`
}

func init() {
	specs, err := loadQuoteRESTSpecs()
	if err != nil {
		panic(err)
	}
	for i := range specs {
		if specs[i].Group == simGroupID {
			continue
		}
		rootCmd.AddCommand(newQuoteRESTCommand(&specs[i]))
	}
}

func loadQuoteRESTSpecs() ([]quoteRESTSpec, error) {
	var specs []quoteRESTSpec
	if err := json.Unmarshal(quoteRESTSpecsJSON, &specs); err != nil {
		return nil, fmt.Errorf("parse quote REST specs: %w", err)
	}
	return specs, nil
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
		usage := quoteRESTUsage(p)
		switch p.Kind {
		case "int":
			cmd.Flags().Int(flagName(p.Name), atoiDefault(p.Default), usage)
		case "bool":
			cmd.Flags().Bool(flagName(p.Name), p.Default == "true", usage)
		case "float":
			cmd.Flags().Float64(flagName(p.Name), atofDefault(p.Default), usage)
		case "strings":
			cmd.Flags().StringSlice(flagName(p.Name), nil, usage)
		default:
			cmd.Flags().String(flagName(p.Name), resolveStaticDefault(p.Default), usage)
		}
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
		return spec.Cmd + " [plate]"
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
		return fmt.Sprintf("  futu %s\n  futu %s FUTU", spec.Cmd, spec.Cmd)
	case "plate":
		return fmt.Sprintf("  futu %s\n  futu %s HK.LIST1045", spec.Cmd, spec.Cmd)
	default:
		return fmt.Sprintf("  futu %s\n  futu %s --format json", spec.Cmd, spec.Cmd)
	}
}

func quoteRESTUsage(p quoteRESTParam) string {
	parts := []string{p.Name}
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
		keyword := "00700"
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
			if parsed, err := parseOneFutuCode(fv); err == nil {
				plate = parsed
			} else {
				plate = fv
			}
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
	if cmd.Flags().Changed("stock-owner") || cmd.Flags().Changed("screen-groups") {
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
	if !cmd.Flags().Changed(name) && p.Default == "" && !p.Required {
		return nil, false, nil
	}
	switch p.Kind {
	case "int":
		v, err := cmd.Flags().GetInt(name)
		if err != nil {
			return nil, false, err
		}
		if !cmd.Flags().Changed(name) && p.Default == "" && v == 0 && !p.Required {
			return nil, false, nil
		}
		return v, true, nil
	case "bool":
		v, err := cmd.Flags().GetBool(name)
		if err != nil {
			return nil, false, err
		}
		if !cmd.Flags().Changed(name) && p.Default == "" && !v && !p.Required {
			return nil, false, nil
		}
		return v, true, nil
	case "float":
		v, err := cmd.Flags().GetFloat64(name)
		if err != nil {
			return nil, false, err
		}
		if !cmd.Flags().Changed(name) && p.Default == "" && v == 0 && !p.Required {
			return nil, false, nil
		}
		return v, true, nil
	case "strings":
		v, err := cmd.Flags().GetStringSlice(name)
		if err != nil {
			return nil, false, err
		}
		if len(v) == 0 {
			return nil, false, nil
		}
		return v, true, nil
	default:
		v, err := cmd.Flags().GetString(name)
		if err != nil {
			return nil, false, err
		}
		v = resolveDateToken(v)
		if v == "" {
			return nil, false, nil
		}
		if p.Kind == "json" {
			var parsed any
			if err := json.Unmarshal([]byte(v), &parsed); err != nil {
				return nil, false, fmt.Errorf("--%s requires JSON: %w", name, err)
			}
			return parsed, true, nil
		}
		return v, true, nil
	}
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
	return strings.ReplaceAll(name, "_", "-")
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
