package contracts

// AUTH01001 indicates invalid credentials.
const AUTH01001 = "AUTH01001"

// AUTH01002 indicates an invalid or expired session.
const AUTH01002 = "AUTH01002"

// AUTH01003 indicates that Google sign-in is unavailable or failed.
const AUTH01003 = "AUTH01003"

// AUTH01004 indicates that registration data is invalid.
const AUTH01004 = "AUTH01004"

// AUTH01005 indicates that the email is already registered.
const AUTH01005 = "AUTH01005"

// AUTH02001 indicates that super-owner access is required.
const AUTH02001 = "AUTH02001"

// USER01001 indicates that a requested user was not found.
const USER01001 = "USER01001"

// USER01002 indicates invalid profile data.
const USER01002 = "USER01002"

// USER02001 indicates an avatar upload failed.
const USER02001 = "USER02001"

// USER02002 indicates an avatar was rejected (unsupported type or too large).
const USER02002 = "USER02002"

// USER03001 indicates the user's search history could not be loaded.
const USER03001 = "USER03001"

// USER03002 indicates the user's search history could not be cleared.
const USER03002 = "USER03002"

// SITE01001 indicates site settings are not available.
const SITE01001 = "SITE01001"

// AISY01001 indicates that the AI provider is unavailable.
const AISY01001 = "AISY01001"

// AISY01002 indicates that an AI request timed out.
const AISY01002 = "AISY01002"

// AISY01003 indicates that the AI provider returned an error.
const AISY01003 = "AISY01003"

// AISY01004 indicates that the submitted question is empty or invalid.
const AISY01004 = "AISY01004"

// AISY02001 indicates that the AI queue is full.
const AISY02001 = "AISY02001"

// AISY02002 indicates that a rate limit was exceeded.
const AISY02002 = "AISY02002"

// AISY02003 indicates that the daily question quota was exceeded.
const AISY02003 = "AISY02003"

// AISY02004 indicates that an anonymous guest already used their single AI
// search allowance and must sign in to continue.
const AISY02004 = "AISY02004"

// AISY03003 indicates an invalid URL.
const AISY03003 = "AISY03003"

// AISY03004 indicates that a page crawl failed.
const AISY03004 = "AISY03004"

// SESS01001 indicates that a chat session was not found.
const SESS01001 = "SESS01001"

// SESS01002 indicates that access to a chat session was denied.
const SESS01002 = "SESS01002"

// AISY03002 indicates a URL was blocked by the SSRF guard.
const AISY03002 = "AISY03002"

// ADMN01001 indicates a user operation failed in the control panel.
const ADMN01001 = "ADMN01001"

// ADMN02001 indicates statistics could not be computed.
const ADMN02001 = "ADMN02001"

// ADMN03001 indicates an AI provider was not found.
const ADMN03001 = "ADMN03001"

// ADMN03002 indicates an invalid AI provider configuration.
const ADMN03002 = "ADMN03002"

// ADMN04001 indicates an invalid AI queue configuration.
const ADMN04001 = "ADMN04001"

// ADMN05001 indicates invalid site settings.
const ADMN05001 = "ADMN05001"

// ADMN05002 indicates a site logo upload failed.
const ADMN05002 = "ADMN05002"

// ADMN05003 indicates a site favicon upload failed.
const ADMN05003 = "ADMN05003"

// ADMN06001 indicates the Ollama server could not be reached or the base URL is invalid.
const ADMN06001 = "ADMN06001"
