package league

// DTOs returned by the service — the typed contract the HTTP layer
// serializes to JSON for the phone. Field names/shapes match the mobile
// page's JS exactly, since that side was kept as-is.

type Champion struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TeamSlot struct {
	Champion int    `json:"champion"`
	Intent   int    `json:"intent"`
	Position string `json:"position"`
	Name     string `json:"name"`
	Me       bool   `json:"me"`
	Locked   bool   `json:"locked"`
}

type Teams struct {
	Allies    []TeamSlot `json:"allies"`
	Enemies   []TeamSlot `json:"enemies"`
	AllyBans  []int      `json:"allyBans"`
	EnemyBans []int      `json:"enemyBans"`
}

type Action struct {
	ID         int    `json:"id"`
	Type       string `json:"type"`
	InProgress bool   `json:"inProgress"`
	ChampionID int    `json:"championId"`
}

type SelectState struct {
	TimeLeft       int     `json:"timeLeft"`
	LockedChampion int     `json:"lockedChampion"`
	Action         *Action `json:"action"`
	Allowed        []int   `json:"allowed"`
	Teams          Teams   `json:"teams"`
}

type StateResult struct {
	Connected  bool         `json:"connected"`
	Phase      string       `json:"phase,omitempty"`
	AutoAccept bool         `json:"autoAccept"`
	Select     *SelectState `json:"select"`
	Error      string       `json:"error,omitempty"`
}

type RunePage struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Editable bool   `json:"editable"`
	Primary  int    `json:"primary"`
	Sub      int    `json:"sub"`
	Perks    []int  `json:"perks"`
}

type Perk struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

type Style struct {
	ID         int     `json:"id"`
	Name       string  `json:"name"`
	Icon       string  `json:"icon"`
	Keystones  []int   `json:"keystones"`
	Rows       [][]int `json:"rows"`
	Shards     [][]int `json:"shards"`
	AllowedSub []int   `json:"allowedSub"`
}

type RunesResult struct {
	Pages     []RunePage      `json:"pages"`
	CurrentID *int            `json:"currentId"`
	Styles    []Style         `json:"styles"`
	Perks     map[string]Perk `json:"perks"`
}

// PageFields is the validated, ready-to-send shape for creating/overwriting
// a rune page.
type PageFields struct {
	Name           string `json:"name"`
	PrimaryStyleID int    `json:"primaryStyleId"`
	SubStyleID     int    `json:"subStyleId"`
	SelectedPerks  []int  `json:"selectedPerkIds"`
}

type RecommendedPage struct {
	Primary  int    `json:"primary"`
	Sub      int    `json:"sub"`
	Perks    []int  `json:"perks"`
	Position string `json:"position"`
}

type RecommendedResult struct {
	Champion int               `json:"champion"`
	Pages    []RecommendedPage `json:"pages"`
}
