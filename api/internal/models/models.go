package models

// User es el perfil expuesto al frontend.
type User struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FullName  string `json:"fullName"`
	AvatarURL string `json:"avatarUrl"`
	Role      string `json:"role"`
	Tier      string `json:"tier"`
	CreatedAt string `json:"createdAt,omitempty"`
}

// Plugin es un producto del catalogo.
type Plugin struct {
	ID           string `json:"id"`
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	RequiredTier string `json:"requiredTier"`
	Active       bool   `json:"active"`
	SortOrder    int    `json:"sortOrder"`
	Package      string `json:"package,omitempty"`
	JarFile      string `json:"jarFile,omitempty"`
	Version      string `json:"version,omitempty"`
	Sha256       string `json:"sha256,omitempty"`
	URL          string `json:"url,omitempty"`
}

// PluginAccess es el estado efectivo de acceso de un usuario a un plugin.
type PluginAccess struct {
	Plugin   Plugin `json:"plugin"`
	Access   bool   `json:"access"`
	Source   string `json:"source"` // tier | granted | revoked
	Override *bool  `json:"override,omitempty"`
}

// MeResponse es la respuesta de GET /api/me.
type MeResponse struct {
	User    User           `json:"user"`
	IsAdmin bool           `json:"isAdmin"`
	Plugins []PluginAccess `json:"plugins"`
}

// AdminUser es una fila del panel de administracion.
type AdminUser struct {
	User         User `json:"user"`
	GrantedCount int  `json:"grantedCount"`
	RevokedCount int  `json:"revokedCount"`
}
