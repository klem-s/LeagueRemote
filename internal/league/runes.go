package league

import (
	"errors"
	"sort"
	"strconv"
	"strings"
)

// runeData loads and caches perk names/icons and styles (trees), in the
// shape the mobile page's editor expects: keystones, 3 main rows, shards,
// with a fallback shard row for styles that define none of their own.
func (s *Service) runeData() (map[int]Perk, []Style, error) {
	s.runeMu.Lock()
	defer s.runeMu.Unlock()

	if s.perksCache == nil {
		data, err := s.client.Get("/lol-perks/v1/perks")
		if err != nil {
			return nil, nil, err
		}
		perks := map[int]Perk{}
		for _, item := range asList(data) {
			p := asMap(item)
			perks[asInt(p["id"])] = Perk{Name: asString(p["name"]), Icon: asString(p["iconPath"])}
		}
		s.perksCache = perks
	}

	if s.stylesCache == nil {
		data, err := s.client.Get("/lol-perks/v1/styles")
		if err != nil {
			return nil, nil, err
		}
		var styles []Style
		for _, item := range asList(data) {
			st := asMap(item)
			var keys []int
			var rows [][]int
			var shards [][]int
			for i, slotRaw := range asList(st["slots"]) {
				slot := asMap(slotRaw)
				t := asString(slot["type"])
				perks := toIntSlice(slot["perks"])
				switch {
				case t == "kKeyStone" || (keys == nil && i == 0):
					keys = perks
				case strings.Contains(t, "StatMod"):
					shards = append(shards, perks)
				default:
					rows = append(rows, perks)
				}
			}
			if len(rows) > 3 {
				rows = rows[:3]
			}
			styles = append(styles, Style{
				ID: asInt(st["id"]), Name: asString(st["name"]), Icon: asString(st["iconPath"]),
				Keystones: nonNilInts(keys), Rows: rows, Shards: shards,
				AllowedSub: toIntSlice(st["allowedSubStyles"]),
			})
		}

		var fallback [][]int
		for _, st := range styles {
			if len(st.Shards) > 0 {
				fallback = st.Shards
				break
			}
		}
		for i := range styles {
			if len(styles[i].Shards) == 0 && fallback != nil {
				styles[i].Shards = fallback
			}
		}

		filtered := []Style{}
		for _, st := range styles {
			if len(st.Keystones) > 0 {
				filtered = append(filtered, st)
			}
		}
		s.stylesCache = filtered
	}

	return s.perksCache, s.stylesCache, nil
}

func nonNilInts(v []int) []int {
	if v == nil {
		return []int{}
	}
	return v
}

func (s *Service) Runes() (RunesResult, error) {
	perks, styles, err := s.runeData()
	if err != nil {
		return RunesResult{}, err
	}
	pagesRaw, err := s.client.Get("/lol-perks/v1/pages")
	if err != nil {
		return RunesResult{}, err
	}
	var currentID *int
	if cp, err := s.client.Get("/lol-perks/v1/currentpage"); err == nil {
		id := asInt(asMap(cp)["id"])
		currentID = &id
	}

	out := []RunePage{}
	for _, item := range asList(pagesRaw) {
		p := asMap(item)
		out = append(out, RunePage{
			ID: asInt(p["id"]), Name: asString(p["name"]), Editable: asBool(p["isEditable"]),
			Primary: asInt(p["primaryStyleId"]), Sub: asInt(p["subStyleId"]), Perks: toIntSlice(p["selectedPerkIds"]),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Editable != out[j].Editable {
			return out[i].Editable
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})

	perksOut := map[string]Perk{}
	for id, p := range perks {
		perksOut[strconv.Itoa(id)] = p
	}

	return RunesResult{Pages: out, CurrentID: currentID, Styles: styles, Perks: perksOut}, nil
}

// PageFieldsFrom validates the raw JSON a client posted (id is handled by
// the caller) into a PageFields ready to send to the LCU.
func PageFieldsFrom(body map[string]interface{}) (PageFields, error) {
	idsRaw, _ := body["selectedPerkIds"].([]interface{})
	if len(idsRaw) != 9 {
		return PageFields{}, &ValidationError{"Page incomplète : il faut 9 runes"}
	}
	ids := make([]int, 9)
	for i, v := range idsRaw {
		ids[i] = asInt(v)
	}
	name := strings.TrimSpace(asString(body["name"]))
	if name == "" {
		name = "LoL Remote"
	}
	if len(name) > 40 {
		name = name[:40]
	}
	return PageFields{
		Name: name, PrimaryStyleID: asInt(body["primaryStyleId"]),
		SubStyleID: asInt(body["subStyleId"]), SelectedPerks: ids,
	}, nil
}

func (f PageFields) toLCU() map[string]interface{} {
	return map[string]interface{}{
		"name": f.Name, "primaryStyleId": f.PrimaryStyleID,
		"subStyleId": f.SubStyleID, "selectedPerkIds": f.SelectedPerks,
	}
}

// SavePage mirrors Python's save_page: PUTs onto an existing editable page
// (recreating it via DELETE+POST if the PUT is rejected, which happens on
// some client versions), or POSTs a new page when pageID is nil.
func (s *Service) SavePage(fields PageFields, pageID *int) error {
	if pageID != nil {
		pagesRaw, err := s.client.Get("/lol-perks/v1/pages")
		if err != nil {
			return err
		}
		var page map[string]interface{}
		for _, item := range asList(pagesRaw) {
			p := asMap(item)
			if asInt(p["id"]) == *pageID {
				page = p
				break
			}
		}
		if page == nil || !asBool(page["isEditable"]) {
			return &ValidationError{"Cette page n'est pas modifiable"}
		}
		merged := map[string]interface{}{}
		for k, v := range page {
			merged[k] = v
		}
		for k, v := range fields.toLCU() {
			merged[k] = v
		}
		merged["current"] = true

		_, err = s.client.Put("/lol-perks/v1/pages/"+strconv.Itoa(*pageID), merged)
		if err == nil {
			_, err = s.client.Put("/lol-perks/v1/currentpage", *pageID)
			return err
		}
		var remoteErr *RemoteError
		if !errors.As(err, &remoteErr) {
			return err
		}
		_, _ = s.client.Delete("/lol-perks/v1/pages/" + strconv.Itoa(*pageID))
	}

	body := fields.toLCU()
	body["current"] = true
	if _, err := s.client.Post("/lol-perks/v1/pages", body); err != nil {
		var remoteErr *RemoteError
		if errors.As(err, &remoteErr) {
			return &ValidationError{"Plus de place pour une nouvelle page. Supprime une page, " +
				"ou active une page perso pour qu'elle soit remplacée."}
		}
		return err
	}
	return nil
}

// ApplyPage applies runes on the active page if it's editable, otherwise
// creates a new one, and returns the page name they ended up on.
func (s *Service) ApplyPage(fields PageFields) (string, error) {
	cur, err := s.client.Get("/lol-perks/v1/currentpage")
	if err != nil {
		return "", err
	}
	curMap := asMap(cur)
	if asBool(curMap["isEditable"]) {
		id := asInt(curMap["id"])
		if err := s.SavePage(fields, &id); err != nil {
			return "", err
		}
		return asString(curMap["name"]), nil
	}
	if err := s.SavePage(fields, nil); err != nil {
		return "", err
	}
	return fields.Name, nil
}

func (s *Service) ActivatePage(id int) error {
	_, err := s.client.Put("/lol-perks/v1/currentpage", id)
	return err
}

func (s *Service) DeletePage(id int) error {
	_, err := s.client.Delete("/lol-perks/v1/pages/" + strconv.Itoa(id))
	return err
}

func (s *Service) Recommended() (RecommendedResult, error) {
	raw, err := s.client.Get("/lol-champ-select/v1/session")
	if err != nil {
		return RecommendedResult{}, err
	}
	sess := asMap(raw)
	me := asInt(sess["localPlayerCellId"])
	cell := map[string]interface{}{}
	for _, c := range asList(sess["myTeam"]) {
		cm := asMap(c)
		if asInt(cm["cellId"]) == me {
			cell = cm
			break
		}
	}
	cid := asInt(cell["championId"])
	if cid == 0 {
		cid = asInt(cell["championPickIntent"])
	}
	if cid == 0 {
		return RecommendedResult{Champion: 0, Pages: []RecommendedPage{}}, nil
	}
	pos := strings.ToUpper(asString(cell["assignedPosition"]))
	if pos == "" {
		pos = "NONE"
	}

	var data []interface{}
	path := "/lol-perks/v1/recommended-pages/champion/" + strconv.Itoa(cid) + "/position/" + pos + "/map/11"
	if rawPages, err := s.client.Get(path); err == nil {
		data = asList(rawPages)
	}

	pages := []RecommendedPage{}
	for _, item := range data {
		r := asMap(item)
		var ids []int
		for _, p := range asList(r["perks"]) {
			if pm, ok := p.(map[string]interface{}); ok {
				ids = append(ids, asInt(pm["id"]))
			} else {
				ids = append(ids, asInt(p))
			}
		}
		primary := asInt(r["primaryPerkStyleId"])
		if len(ids) >= 9 && primary != 0 {
			pages = append(pages, RecommendedPage{
				Primary: primary, Sub: asInt(r["secondaryPerkStyleId"]),
				Perks: ids[:9], Position: asString(r["position"]),
			})
		}
	}
	return RecommendedResult{Champion: cid, Pages: pages}, nil
}
