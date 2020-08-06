package Group

type Group struct {
	GroupName        string `json:"group_name"`
	GroupDescription string `json:"group_description"`
	GroupID          int    `json:"group_id"`
	ParentID         int    `json:"parent_id"`
}
