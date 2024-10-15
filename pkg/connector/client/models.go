package client

import "time"

type AICCLaunch struct {
	Params string `json:"params"`
	Url    string `json:"url"`
}
type ApprovalManager struct {
	Id             string `json:"id"`
	Email          string `json:"email"`
	ExternalUserId string `json:"externalUserId"`
}

type Associations struct {
	Areas              []string           `json:"areas"`
	Subjects           []string           `json:"subjects"`
	Channels           []Channel          `json:"channels"`
	LocalizedChannels  []LocalizedChannel `json:"localizedChannels"`
	LicensedLocales    string             `json:"licensedLocales"`
	Skills             []Skill            `json:"skills"`
	Journeys           []Journey          `json:"journeys"`
	Parent             Parent             `json:"parent"`
	TranslationGroupId string             `json:"translationGroupId"`
}

type Channel struct {
	Id    string `json:"id"`
	Link  string `json:"link"`
	Title string `json:"title"`
}

type Characteristics struct {
	EarnsBadge    bool `json:"earnsBadge"`
	HasAssessment bool `json:"hasAssessment"`
	IsCompliance  bool `json:"isCompliance"`
}

type ContentType struct {
	Category     string `json:"category"`
	DisplayLabel string `json:"displayLabel"`
	PercipioType string `json:"percipioType"`
	Source       string `json:"source"`
}

type Course struct {
	AICCLaunch                 AICCLaunch                 `json:"aiccLaunch"`
	AlternateImageUrl          string                     `json:"alternateImageUrl"`
	Associations               Associations               `json:"associations"`
	By                         []string                   `json:"by"`
	Characteristics            Characteristics            `json:"characteristics"`
	Code                       string                     `json:"code"`
	ContentType                ContentType                `json:"contentType"`
	Credentials                Credentials                `json:"credentials"`
	Duration                   string                     `json:"duration"`
	ExpertiseLevels            []string                   `json:"expertiseLevels"`
	Id                         string                     `json:"id"`
	ImageUrl                   string                     `json:"imageUrl"`
	Keywords                   []string                   `json:"keywords"`
	LearningObjectives         []string                   `json:"learningObjectives"`
	Lifecycle                  Lifecycle                  `json:"lifecycle"`
	Link                       string                     `json:"link"`
	LocaleCodes                []string                   `json:"localeCodes"`
	LocalizedMetadata          []LocalizedMetadata        `json:"localizedMetadata"`
	Modalities                 []string                   `json:"modalities"`
	ProviderSpecificAttributes ProviderSpecificAttributes `json:"providerSpecificAttributes"`
	Publication                Publication                `json:"publication"`
	Technologies               []Technology               `json:"technologies"`
	XApiActivityId             string                     `json:"xapiActivityId"`
	XApiActivityTypeId         string                     `json:"xapiActivityTypeId"`
}

type Credentials struct {
	// Continuing Professional Education credits. Possible values include 1.5.
	CpeCredits float64 `json:"cpeCredits"`
	// National Association of State Boards of Accountancy. Seems like a CPA thing.
	NasbaReady bool `json:"nasbaReady"`
	// Professional Development Unit credits. Possible values include 0.75.
	PduCredits float64 `json:"pduCredits"`
}

type CustomAttribute struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Journey struct {
	Id    string `json:"id"`
	Title string `json:"title"`
	Link  string `json:"link"`
}

type Lifecycle struct {
	IncludedFromActivity  bool      `json:"includedFromActivity"`
	LastUpdatedDate       time.Time `json:"lastUpdatedDate"`
	PlannedRetirementDate time.Time `json:"plannedRetirementDate"`
	PublishDate           time.Time `json:"publishDate"`
	RetiredDate           time.Time `json:"retiredDate"`
	Status                string    `json:"status"`
}

type LocalizedChannel struct {
	Description    string `json:"description"`
	Id             string `json:"id"`
	Link           string `json:"link"`
	LocaleCode     string `json:"localeCode"`
	Title          string `json:"title"`
	XapiActivityId string `json:"xapiActivityId"`
}

type LocalizedMetadata struct {
	Description string `json:"description"`
	LocaleCode  string `json:"localeCode"`
	Title       string `json:"title"`
}

type Parent struct {
	Id    string `json:"id"`
	Link  string `json:"link"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

type ProviderSpecificAttributes struct {
	ProviderAssetId   string `json:"providerAssetId"`
	SkillCourseNumber string `json:"skillCourseNumber"`
}

type Publication struct {
	CopyrightYear int    `json:"copyrightYear"`
	Isbn          string `json:"isbn"`
	Publisher     string `json:"publisher"`
}

type Report []ReportEntry

type ReportConfigurations struct {
	Audience                string                `json:"audience,omitempty"`
	ContentType             string                `json:"contentType,omitempty"`
	CsvPreferences          *ReportCsvPreferences `json:"csvPreferences,omitempty"`
	Encrypt                 bool                  `json:"encrypt,omitempty"`
	End                     time.Time             `json:"end,omitempty"`
	FileMask                string                `json:"fileMask,omitempty"`
	FolderName              string                `json:"folderName,omitempty"`
	FormatType              string                `json:"formatType,omitempty"`
	IncludeMillisInFilename bool                  `json:"includeMillisInFilename,omitempty"`
	IsFileRequiredInSftp    bool                  `json:"isFileRequiredInSftp,omitempty"`
	IsPgpFileExtnNotReqrd   bool                  `json:"isPgpFileExtnNotReqrd,omitempty"`
	Locale                  string                `json:"locale,omitempty"`
	Mapping                 string                `json:"mapping,omitempty"`
	Plugin                  string                `json:"plugin,omitempty"`
	SftpId                  string                `json:"sftpId,omitempty"`
	Sort                    *ReportSort           `json:"sort,omitempty"`
	Start                   time.Time             `json:"start,omitempty"`
	Status                  string                `json:"status,omitempty"`
	Template                string                `json:"template,omitempty"`
	TimeFrame               string                `json:"timeFrame,omitempty"`
	TransformName           string                `json:"transformName,omitempty"`
	Zip                     bool                  `json:"zip,omitempty"`
}

type ReportCsvPreferences struct {
	ColumnDelimiter    string `json:"columnDelimiter,omitempty"`
	Header             bool   `json:"header,omitempty"`
	HeaderForNoRecords bool   `json:"headerForNoRecords,omitempty"`
	RowDelimiter       string `json:"rowDelimiter,omitempty"`
}

type ReportEntry struct {
	Audience             string    `json:"audience"`
	BusinessUnit         string    `json:"businessUnit"`
	CompletedDate        time.Time `json:"completedDate"`
	ContentTitle         string    `json:"contentTitle"`
	ContentType          string    `json:"contentType"`
	ContentUUID          string    `json:"contentUuid"`
	CostCenterCode       string    `json:"costCenterCode"`
	CountryName          string    `json:"countryName"`
	DepartmentCode       string    `json:"departmentCode"`
	DepartmentOwner      string    `json:"departmentOwner"`
	DeptName             string    `json:"deptName"`
	DirectManagerName    string    `json:"directManagerName"`
	Division             string    `json:"division"`
	DivisionCode         string    `json:"divisionCode"`
	DivisionOwner        string    `json:"divisonOwner"` // [SIC]
	DurationHms          string    `json:"durationHms"`
	EmailAddress         string    `json:"emailAddress"`
	EmployeeClass        string    `json:"employeeClass"`
	EmployeeId           string    `json:"employeeId"`
	EstimatedDurationHms string    `json:"estimatedDurationHms"`
	FirstAccess          time.Time `json:"firstAccess"`
	FirstName            string    `json:"firstName"`
	Geo                  string    `json:"geo"`
	HireDate             string    `json:"hireDate"`
	HrbpOwner            string    `json:"hrbpOwner"`
	IsAManager           string    `json:"isAManager"`
	LanguageCode         string    `json:"languageCode"`
	LastName             string    `json:"lastName"`
	ManagerEmail         string    `json:"managerEmail"`
	ManagerId            string    `json:"managerId"`
	Status               string    `json:"status"`
	UserId               string    `json:"userId"`
	UserStatus           string    `json:"userStatus"`
	UserUUID             string    `json:"userUuid"`
}

type ReportSort struct {
	Field string `json:"field,omitempty"`
	Order string `json:"order,omitempty"`
}

type ReportStatus struct {
	Id     string `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type Skill struct {
	LocaleCode string   `json:"localeCode"`
	Skills     []string `json:"skills"`
}

type Technology struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type User struct {
	Id                             string            `json:"id"`
	ApprovalManager                ApprovalManager   `json:"approvalManager"`
	CustomAttributes               []CustomAttribute `json:"customAttributes"`
	Email                          string            `json:"email"`
	ExternalUserId                 string            `json:"externalUserId"`
	FirstName                      string            `json:"firstName"`
	HasCoaching                    bool              `json:"hasCoaching"`
	HasEnterpriseCoaching          bool              `json:"hasEnterpriseCoaching"`
	HasEnterpriseCoachingDashboard bool              `json:"hasEnterpriseCoachingDashboard"`
	IsActive                       bool              `json:"isActive"`
	IsInstructor                   bool              `json:"isInstructor"`
	JobTitle                       string            `json:"jobTitle"`
	LastName                       string            `json:"lastName"`
	LoginName                      string            `json:"loginName"`
	Role                           string            `json:"role"`
	UpdatedAt                      string            `json:"updatedAt"`
}
