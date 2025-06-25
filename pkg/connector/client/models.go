package client

import "time"

// AICCLaunch struct represents the AICC launch parameters for a course.
// It is used by the `Course` struct for legacy course content.
// It holds fields such as `Params` and `Url` for AICC course launching.
// This structure organizes data required to launch AICC-compliant e-learning content.
// Instances are populated from the Percipio API response for catalog content.
type AICCLaunch struct {
	Params string `json:"params"`
	Url    string `json:"url"`
}

// ApprovalManager struct represents a user's approval manager.
// It is used by the `User` struct to define a managerial relationship.
// It holds fields such as `Id`, `Email`, and `ExternalUserId`.
// This structure organizes the identity information of a manager.
// Instances are populated from the Percipio API response for user data.
type ApprovalManager struct {
	Id             string `json:"id"`
	Email          string `json:"email"`
	ExternalUserId string `json:"externalUserId"`
}

// Associations struct represents the relationships a course has with other content.
// It is used by the `Course` struct to link to related learning items.
// It holds fields such as `Channels`, `Journeys`, and `Parent`.
// This structure organizes the navigational and relational context of a course.
// Instances are populated from the Percipio API response for catalog content.
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

// Channel struct represents a learning channel.
// It is used by the `Associations` struct.
// It holds fields such as `Id`, `Link`, and `Title`.
// This structure organizes basic information about a learning channel.
// Instances are populated from the Percipio API response for catalog content.
type Channel struct {
	Id    string `json:"id"`
	Link  string `json:"link"`
	Title string `json:"title"`
}

// Characteristics struct represents boolean properties of a course.
// It is used by the `Course` struct.
// It holds fields such as `EarnsBadge` and `HasAssessment`.
// This structure organizes key flags that describe the nature of a course.
// Instances are populated from the Percipio API response for catalog content.
type Characteristics struct {
	EarnsBadge    bool `json:"earnsBadge"`
	HasAssessment bool `json:"hasAssessment"`
	IsCompliance  bool `json:"isCompliance"`
}

// ContentType struct represents the type of a learning content item.
// It is used by the `Course` struct.
// It holds fields such as `Category`, `PercipioType`, and `Source`.
// This structure organizes the classification details of a content item.
// Instances are populated from the Percipio API response for catalog content.
type ContentType struct {
	Category     string `json:"category"`
	DisplayLabel string `json:"displayLabel"`
	PercipioType string `json:"percipioType"`
	Source       string `json:"source"`
}

// Course struct represents a single learning content item, like a course or assessment.
// It is the primary data representation for content resources synced by the connector.
// It holds fields such as `Id`, `ContentType`, `Lifecycle`, and `LocalizedMetadata`.
// This structure organizes all metadata related to a piece of learning content.
// Instances are created from the Percipio API response when fetching catalog content or searching for content.
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

// Credentials struct represents any professional credits associated with a course.
// It is used by the `Course` struct.
// It holds fields such as `CpeCredits` and `PduCredits`.
// This structure organizes professional education credit information.
// Instances are populated from the Percipio API response for catalog content.
type Credentials struct {
	// Continuing Professional Education credits. Possible values include 1.5.
	CpeCredits float64 `json:"cpeCredits"`
	// National Association of State Boards of Accountancy. Seems like a CPA thing.
	NasbaReady bool `json:"nasbaReady"`
	// Professional Development Unit credits. Possible values include 0.75.
	PduCredits float64 `json:"pduCredits"`
}

// CustomAttribute struct represents a custom user attribute.
// It is used by the `User` struct.
// It holds fields such as `Id`, `Name`, and `Value`.
// This structure organizes key-value pairs for custom user data.
// Instances are populated from the Percipio API response for user data.
type CustomAttribute struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Journey struct represents a learning journey or path.
// It is used by the `Associations` struct.
// It holds fields such as `Id`, `Title`, and `Link`.
// This structure organizes basic information about a learning journey.
// Instances are populated from the Percipio API response for catalog content.
type Journey struct {
	Id    string `json:"id"`
	Title string `json:"title"`
	Link  string `json:"link"`
}

// Lifecycle struct represents the publication status and dates for a course.
// It is used by the `Course` struct.
// It holds fields such as `Status`, `PublishDate`, and `RetiredDate`.
// This structure organizes the publication lifecycle of a content item.
// Instances are populated from the Percipio API response for catalog content.
type Lifecycle struct {
	IncludedFromActivity  bool      `json:"includedFromActivity"`
	LastUpdatedDate       time.Time `json:"lastUpdatedDate"`
	PlannedRetirementDate time.Time `json:"plannedRetirementDate"`
	PublishDate           time.Time `json:"publishDate"`
	RetiredDate           time.Time `json:"retiredDate"`
	Status                string    `json:"status"`
}

// LocalizedChannel struct represents language-specific details for a channel.
// It is used by the `Associations` struct.
// It holds fields such as `Title`, `Description`, and `LocaleCode`.
// This structure organizes translated metadata for a learning channel.
// Instances are populated from the Percipio API response for catalog content.
type LocalizedChannel struct {
	Description    string `json:"description"`
	Id             string `json:"id"`
	Link           string `json:"link"`
	LocaleCode     string `json:"localeCode"`
	Title          string `json:"title"`
	XapiActivityId string `json:"xapiActivityId"`
}

// LocalizedMetadata struct represents language-specific metadata for a content item.
// It is used by the `Course` struct.
// It holds fields such as `Title`, `Description`, and `LocaleCode`.
// This structure organizes translated text for course details.
// Instances are populated from the Percipio API response for catalog content.
type LocalizedMetadata struct {
	Description string `json:"description"`
	LocaleCode  string `json:"localeCode"`
	Title       string `json:"title"`
}

// Parent struct represents the parent content item in a hierarchy.
// It is used by the `Associations` struct.
// It holds fields such as `Id`, `Title`, and `Type`.
// This structure organizes information about a parent learning object.
// Instances are populated from the Percipio API response for catalog content.
type Parent struct {
	Id    string `json:"id"`
	Link  string `json:"link"`
	Title string `json:"title"`
	Type  string `json:"type"`
}

// ProviderSpecificAttributes struct holds attributes specific to the content provider.
// It is used by the `Course` struct.
// It holds fields such as `ProviderAssetId`.
// This structure organizes provider-specific identifiers.
// Instances are populated from the Percipio API response for catalog content.
type ProviderSpecificAttributes struct {
	ProviderAssetId   string `json:"providerAssetId"`
	SkillCourseNumber string `json:"skillCourseNumber"`
}

// Publication struct represents publication details for a content item.
// It is used by the `Course` struct.
// It holds fields such as `Publisher` and `CopyrightYear`.
// This structure organizes publication metadata, similar to a book.
// Instances are populated from the Percipio API response for catalog content.
type Publication struct {
	CopyrightYear int    `json:"copyrightYear"`
	Isbn          string `json:"isbn"`
	Publisher     string `json:"publisher"`
}

// Report is a type alias for a slice of `ReportEntry`, representing the full learning activity report.
// It is used by the client to hold the results of a generated report.
// It is a collection of `ReportEntry` structs, where each entry is a row in the report.
// This structure organizes the raw report data before it is processed into the `StatusesStore` cache.
// Instances are populated by unmarshaling the JSON array from the report download endpoint.
type Report []ReportEntry

// ReportConfigurations struct defines the parameters for requesting a new report.
// It is used by the `GenerateLearningActivityReport` function to specify the report's scope.
// It holds fields such as `Template`, `Start`, and `End` to define the report type and time frame.
// This structure organizes the configuration options for a new report request.
// Instances are created and serialized as the JSON body of a POST request to the reporting API.
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

// ReportCsvPreferences struct defines CSV formatting options for a report.
// It is used by the `ReportConfigurations` struct.
// It holds fields such as `ColumnDelimiter` and `Header`.
// This structure organizes CSV-specific settings within a report request.
// Instances are created as part of a `ReportConfigurations` object.
type ReportCsvPreferences struct {
	ColumnDelimiter    string `json:"columnDelimiter,omitempty"`
	Header             bool   `json:"header,omitempty"`
	HeaderForNoRecords bool   `json:"headerForNoRecords,omitempty"`
	RowDelimiter       string `json:"rowDelimiter,omitempty"`
}

// ReportEntry struct represents a single row in a learning activity report.
// It is the basic unit of the `Report` slice.
// It holds fields such as `UserUUID`, `ContentUUID`, and `Status`.
// This structure organizes the data for a single user-to-content interaction.
// Instances are created by unmarshaling the JSON array from the report download endpoint.
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
	DivisionOwner        string    `json:"divisonOwner"`
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

// ReportSort struct defines sorting options for a report.
// It is used by the `ReportConfigurations` struct.
// It holds fields `Field` and `Order` to specify a sort column and direction.
// This structure organizes sorting parameters within a report request.
// Instances are created as part of a `ReportConfigurations` object.
type ReportSort struct {
	Field string `json:"field,omitempty"`
	Order string `json:"order,omitempty"`
}

// ReportStatus struct represents the status of an asynchronous report generation job.
// It is used by the client to track the progress of a report.
// It holds fields `Id`, `Status`, and `Error`.
// This structure organizes the state of a background reporting job.
// Instances are populated from the API response when a report is first generated and during polling.
type ReportStatus struct {
	Id     string `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Skill struct represents a skill tag.
// It is used by the `Associations` struct.
// It holds a `LocaleCode` and a slice of `Skills` strings.
// This structure organizes skill information associated with a content item.
// Instances are populated from the Percipio API response for catalog content.
type Skill struct {
	LocaleCode string   `json:"localeCode"`
	Skills     []string `json:"skills"`
}

// Technology struct represents a technology tag associated with a content item.
// It is used by the `Course` struct.
// It holds `Title` and `Version` of a technology.
// This structure organizes technology and version information related to a course.
// Instances are populated from the Percipio API response for catalog content.
type Technology struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

// User struct represents a single user identity in Percipio.
// It is the primary data representation for user principals synced by the connector.
// It holds fields such as `Id`, `Email`, `LoginName`, and `Role`.
// This structure organizes all metadata related to a user.
// Instances are created from the Percipio API response when fetching user data.
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
