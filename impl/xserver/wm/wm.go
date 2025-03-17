package wm

import (
	"fmt"
	"sort"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/wiedzmin/toolbox/impl"
	"github.com/wiedzmin/toolbox/impl/redis"
	"go.uber.org/zap"
)

type Keys []string

type Workspaces struct {
	data   []byte
	parsed map[string]string
}

type Keybinding struct {
	Cmd             string `json:"cmd"`
	Key             Keys   `json:"key"`
	Mode            string `json:"mode"`
	LeaveFullscreen bool   `json:"leaveFullscreen"`
	Raw             bool   `json:"raw"`
}

type KeybindingFormattedParts struct {
	Cmd             string
	Key             string
	Mode            string
	LeaveFullscreen string
	Raw             string
	Dangling        string
}

type Keybindings struct {
	data         []byte
	parsed       []Keybinding
	modeBindings map[string]Keys
	partsHelper  map[string]KeybindingFormattedParts
	modeNames    []string
}

type Modebinding struct {
	Name     string
	Prefix   Keys
	Dangling bool
}

type Modebindings struct {
	data        []byte
	parsed      map[string]Keys
	keyBindings []Keybinding
	names       []string
}

type FnFormatKBParts func(*Keybinding, *Modebinding) KeybindingFormattedParts

type FnFormatKBPartsStr func(KeybindingFormattedParts) string

var (
	logger *zap.Logger
	r      *redis.Client
)

const (
	keyNameDangling = "dangling"
	keyNameRoot     = "root"
	treeTextIndent  = 4
)

func init() {
	var err error
	logger = impl.NewLogger()
	r, err = redis.NewRedisLocal()
	if err != nil {
		l := logger.Sugar()
		l.Fatalw("[init]", "failed connecting to Redis", err)
	}
}

type ErrLinkBroken struct {
	Reason string
}

func (e ErrLinkBroken) Error() string {
	return e.Reason
}

func NewWorkspaces(data []byte) (*Workspaces, error) {
	var result Workspaces
	result.data = data
	err := jsoniter.Unmarshal(data, &result.parsed)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func WorkspacesFromRedis(key string) (*Workspaces, error) {
	workspacesData, err := r.GetValue(key)
	if err != nil {
		return nil, err
	}
	return NewWorkspaces(workspacesData)
}

func (wss *Workspaces) Fuzzy() []string {
	var result []string
	for ws, key := range wss.parsed {
		result = append(result, fmt.Sprintf("%-30s | %-10s", ws, key))
	}
	return result
}

func (wss *Workspaces) AsText() string {
	return strings.Join(wss.Fuzzy()[:], "\n")
}

func NewModebindings(data []byte) (*Modebindings, error) {
	var result Modebindings
	result.data = data
	err := jsoniter.Unmarshal(data, &result.parsed)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func ModebindingsFromRedis(key string) (*Modebindings, error) {
	modebindingsData, err := r.GetValue(key)
	if err != nil {
		return nil, err
	}
	return NewModebindings(modebindingsData)
}

func (m *Modebindings) SetKeybindings(kb *Keybindings) {
	m.keyBindings = kb.parsed
}

func GetModebinding(mb map[string]Keys, mode string) Modebinding {
	var mbinding Modebinding

	if mode == "root" {
		mbinding.Name = "root"
		mbinding.Dangling = false
	} else {
		prefix, ok := mb[mode]
		if ok {
			mbinding.Name = mode
			mbinding.Prefix = prefix
			mbinding.Dangling = false
		} else {
			mbinding.Dangling = true
		}
	}

	return mbinding
}

func (mb *Modebindings) Items() []Modebinding {
	var result []Modebinding
	for mode, prefix := range mb.parsed {
		result = append(result, Modebinding{
			Name:   mode,
			Prefix: prefix,
		})
	}
	return result
}

func (mb *Modebindings) Fuzzy() []string {
	var result []string
	for _, b := range mb.Items() {
		result = append(result, b.Format())
	}
	return result
}

func (mb *Modebindings) AsText() string {
	return strings.Join(mb.Fuzzy()[:], "\n")
}

func NewKeybindings(data []byte) (*Keybindings, error) {
	var result Keybindings
	result.data = data
	err := jsoniter.Unmarshal(data, &result.parsed)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func KeybindingsFromRedis(key string) (*Keybindings, error) {
	keybindingsData, err := r.GetValue(key)
	if err != nil {
		return nil, err
	}
	return NewKeybindings(keybindingsData)
}

func (kb *Keybindings) SetModebindings(m *Modebindings) {
	kb.modeBindings = m.parsed
}

func (kb *Keybindings) Items(fnFormat FnFormatKBParts, sortModes bool) (map[string][]KeybindingFormattedParts, error) {
	if kb.modeBindings == nil {
		return nil, ErrLinkBroken{Reason: "mode bindings metadata is not linked"}
	}

	if len(kb.modeNames) == 0 {
		for mode := range kb.modeBindings {
			kb.modeNames = append(kb.modeNames, mode)
		}
		if sortModes {
			sort.Strings(kb.modeNames)
		}
	}

	result := make(map[string][]KeybindingFormattedParts)

	var key string
	for _, meta := range kb.parsed {
		mb := GetModebinding(kb.modeBindings, meta.Mode)
		if mb.Name == "root" {
			key = keyNameRoot
		} else if mb.Dangling {
			key = keyNameDangling
		} else {
			key = meta.Mode
		}
		_, ok := result[key]
		if !ok {
			kbs := make([]KeybindingFormattedParts, 0, 10)
			result[key] = kbs
		}
		result[key] = append(result[key], fnFormat(&meta, &mb))
	}

	return result, nil
}

func (kb *Keybindings) GetPartsForSelection(selection string) (*KeybindingFormattedParts, error) {
	parts, ok := kb.partsHelper[selection]
	if !ok {
		return nil, fmt.Errorf("no parts found for selection")
	}

	return &parts, nil
}

func FormatPartsCommon(k *Keybinding, mb *Modebinding) KeybindingFormattedParts {
	var result KeybindingFormattedParts

	var keysSlice []string

	if mb != nil && mb.Name != "root" {
		keysSlice = append(keysSlice, mb.Prefix.Format())
	}
	keysSlice = append(keysSlice, k.Key.Format())
	result.Key = strings.Join(keysSlice, " ")

	cmd := k.Cmd
	if strings.Contains(cmd, "/nix/store") {
		parts := strings.Split(cmd, "/bin/")
		cmd = parts[len(parts)-1]
	}
	result.Cmd = cmd

	result.Mode = k.Mode

	if k.LeaveFullscreen {
		result.LeaveFullscreen = "yes"
	} else {
		result.LeaveFullscreen = "no"
	}
	if k.Raw {
		result.Raw = "yes"
	} else {
		result.Raw = "no"
	}

	if mb.Dangling {
		result.Dangling = "yes"
	} else {
		result.Dangling = "no"
	}

	return result
}

func FormatFuzzyCommon(parts KeybindingFormattedParts) string {
	return fmt.Sprintf("%-30s | %-10s | %s", parts.Key, parts.Mode, parts.Cmd)
}

func FormatTextFlat(parts KeybindingFormattedParts) string {
	return fmt.Sprintf("%-30s | %-10s | leave fullscreen: %-3s | raw: %-3s | dangling: %s | %s ",
		parts.Key, parts.Mode, parts.LeaveFullscreen, parts.Raw, parts.Dangling, parts.Cmd)
}

func FormatTextIndented(parts KeybindingFormattedParts) string {
	// FIXME: parameterize initial indentation
	return fmt.Sprintf("    %-30s | %-10s | leave fullscreen: %-3s | raw: %-3s | %s ",
		parts.Key, parts.Mode, parts.LeaveFullscreen, parts.Raw, parts.Cmd)
}

func (kb *Keybindings) Fuzzy(fnFormat FnFormatKBPartsStr) ([]string, error) {
	kbItems, err := kb.Items(FormatPartsCommon, true)
	if err != nil {
		return nil, err
	}

	var result []string
	var formattedStr string
	kb.partsHelper = make(map[string]KeybindingFormattedParts)

	partsRoot := kbItems[keyNameRoot]
	partsDangling, okDangling := kbItems[keyNameDangling]
	for _, mode := range kb.modeNames {
		partsMode, okMode := kbItems[mode]
		if okMode {
			for _, parts := range partsMode {
				formattedStr = fnFormat(parts)
				kb.partsHelper[formattedStr] = parts
				result = append(result, formattedStr)
			}
		}
	}
	for _, parts := range partsRoot {
		formattedStr = fnFormat(parts)
		kb.partsHelper[formattedStr] = parts
		result = append(result, formattedStr)
	}
	if okDangling {
		for _, parts := range partsDangling {
			formattedStr = fnFormat(parts)
			kb.partsHelper[formattedStr] = parts
			result = append(result, formattedStr)
		}
	}

	return result, nil
}

func (kb *Keybindings) AsText(fnFormat FnFormatKBPartsStr) (*string, error) {
	kbItems, err := kb.Items(FormatPartsCommon, true)
	if err != nil {
		return nil, err
	}

	var acc []string

	partsRoot := kbItems[keyNameRoot]
	partsDangling, okDangling := kbItems[keyNameDangling]
	for _, mode := range kb.modeNames {
		partsMode, okMode := kbItems[mode]
		if okMode {
			for _, parts := range partsMode {
				acc = append(acc, fnFormat(parts))
			}
		}
	}
	for _, parts := range partsRoot {
		acc = append(acc, fnFormat(parts))
	}
	if okDangling {
		for _, parts := range partsDangling {
			acc = append(acc, fnFormat(parts))
		}
	}

	result := strings.Join(acc[:], "\n")
	return &result, nil
}

func (kb *Keybindings) AsTextTree(fnFormat FnFormatKBPartsStr) (*string, error) {
	kbItems, err := kb.Items(FormatPartsCommon, true)
	if err != nil {
		return nil, err
	}

	var acc []string

	partsRoot := kbItems[keyNameRoot]
	partsDangling, okDangling := kbItems[keyNameDangling]
	for _, mode := range kb.modeNames {
		partsMode, okMode := kbItems[mode]
		if okMode {
			acc = append(acc, mode)
			for _, parts := range partsMode {
				acc = append(acc, fnFormat(parts))
			}
		}
	}
	acc = append(acc, keyNameRoot) // FIXME: consider upcasing
	for _, parts := range partsRoot {
		acc = append(acc, fnFormat(parts))
	}
	if okDangling {
		acc = append(acc, keyNameDangling)
		for _, parts := range partsDangling {
			acc = append(acc, fnFormat(parts))
		}
	}

	result := strings.Join(acc[:], "\n")
	return &result, nil
}

func (mb *Modebinding) Format() string {
	return fmt.Sprintf("%s --> %s", mb.Prefix.Format(), mb.Name)
}

func (ks *Keys) Format() string {
	return strings.Join([]string(*ks), "+")
}

func LinkBindings(kb *Keybindings, m *Modebindings) {
	kb.SetModebindings(m)
	m.SetKeybindings(kb)
}
