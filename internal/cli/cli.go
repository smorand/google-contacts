// Package cli provides the command-line interface for google-contacts.
package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"google-contacts/internal/contacts"
)

// Version information
const Version = "0.1.0"

// RootCmd is the root command for the CLI.
var RootCmd = &cobra.Command{
	Use:   "google-contacts",
	Short: "Google Contacts Manager - Manage Google Contacts",
	Long:  "Create, search, and manage Google Contacts using Google People API v1",
}

// Create command flags
var (
	createFirstName string
	createLastName  string
	createPhones    []string // Multiple phones in format "type:number" or just "number"
	createEmails    []string // Multiple emails in format "type:email" or just "email"
	createCompany   string
	createPosition  string
	createNotes     string
)

// Delete command flags
var (
	deleteForce bool
)

// Update command flags
var (
	updateFirstName  string
	updateLastName   string
	updatePhone      string   // Backward compatible: replaces first phone
	updatePhones     []string // Replaces all phones
	updateAddPhones  []string // Add phones without removing existing
	updateRemPhones  []string // Remove phones by value
	updateEmail      string   // Backward compatible: replaces first email
	updateEmails     []string // Replaces all emails
	updateAddEmails  []string // Add emails without removing existing
	updateRemEmails  []string // Remove emails by value
	updateCompany    string
	updatePosition   string
	updateNotes      string
)

// Command definitions
var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("google-contacts version %s\n", Version)
		},
	}

	createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a new contact",
		Long: `Create a new contact in Google Contacts.

Required fields:
  --firstname, -f: First name
  --lastname, -l:  Last name
  --phone, -p:     Phone number (can be repeated for multiple phones)

Phone number format:
  - Simple: +33612345678 (defaults to "mobile" type)
  - With type: mobile:+33612345678
  - Multiple: -p "mobile:+33612345678" -p "work:+33123456789"

Phone types: mobile (default), work, home, main, other

Email format:
  - Simple: john@acme.com (defaults to "work" type)
  - With type: work:john@acme.com
  - Multiple: -e "work:john@acme.com" -e "home:john@gmail.com"

Email types: work (default), home, other

Recommended fields:
  --company, -c:   Company name

Optional fields:
  --email, -e:     Email address (can be repeated for multiple emails)
  --position, -r:  Role/position at company
  --notes, -n:     Notes about the contact`,
		Example: `  # Create contact with single phone (defaults to mobile)
  google-contacts create -f John -l Doe -p +33612345678

  # Create contact with typed phone
  google-contacts create -f John -l Doe -p "work:+33123456789"

  # Create contact with multiple phones
  google-contacts create -f John -l Doe -p "mobile:+33612345678" -p "work:+33123456789"

  # Create contact with single email (defaults to work)
  google-contacts create -f John -l Doe -p +33612345678 -e john@acme.com

  # Create contact with multiple emails
  google-contacts create -f John -l Doe -p +33612345678 -e "work:john@acme.com" -e "home:john@gmail.com"

  # Create contact with all fields
  google-contacts create -f John -l Doe -p +33612345678 -c "Acme Inc" -r "CTO" -e john@acme.com -n "Met at conference"`,
		RunE: runCreate,
	}

	searchCmd = &cobra.Command{
		Use:   "search <query>",
		Short: "Search contacts",
		Long: `Search for contacts matching the given query.

The query matches on:
  - Names (first name, last name, display name)
  - Email addresses
  - Phone numbers
  - Company names

Output behavior:
  - Multiple results: Shows a summary table
  - Single result: Shows full contact details`,
		Example: `  # Search by name
  google-contacts search "John"

  # Search by partial name
  google-contacts search "Joh"

  # Search by company
  google-contacts search "Acme"

  # Search by phone (partial)
  google-contacts search "0612"`,
		Args: cobra.ExactArgs(1),
		RunE: runSearch,
	}

	showCmd = &cobra.Command{
		Use:   "show <contact-id>",
		Short: "Show contact details",
		Long: `Display full information for a contact.

The contact ID can be:
  - Full resource name: people/c123456789
  - Just the ID: c123456789

Displays:
  - Name (first and last)
  - All phone numbers with labels (mobile, work, home, etc.)
  - All email addresses with labels
  - Company and position
  - Notes
  - Google Contact ID
  - Last update time (if available)`,
		Example: `  # Show by full resource name
  google-contacts show people/c123456789

  # Show by ID only
  google-contacts show c123456789`,
		Args: cobra.ExactArgs(1),
		RunE: runShow,
	}

	deleteCmd = &cobra.Command{
		Use:   "delete <contact-id>",
		Short: "Delete a contact",
		Long: `Delete a contact from Google Contacts.

The contact ID can be:
  - Full resource name: people/c123456789
  - Just the ID: c123456789

Safety:
  - By default, displays contact summary and prompts for confirmation
  - Use --force to skip confirmation

Note: Deletion is permanent and cannot be undone.`,
		Example: `  # Delete with confirmation prompt
  google-contacts delete c123456789

  # Delete without confirmation (use with caution)
  google-contacts delete c123456789 --force`,
		Args: cobra.ExactArgs(1),
		RunE: runDelete,
	}

	updateCmd = &cobra.Command{
		Use:   "update <contact-id>",
		Short: "Update a contact",
		Long: `Update an existing contact in Google Contacts.

The contact ID can be:
  - Full resource name: people/c123456789
  - Just the ID: c123456789

Only the specified fields will be updated. Unspecified fields remain unchanged.

Phone management options:
  --phone, -p:       Update primary phone (replaces first phone)
  --phones:          Replace ALL phones (can be repeated)
  --add-phone:       Add a phone without removing existing (can be repeated)
  --remove-phone:    Remove a phone by value (can be repeated)

Phone format: "type:number" or just "number" (defaults to mobile)
Phone types: mobile (default), work, home, main, other

Email management options:
  --email, -e:       Update primary email (replaces first email)
  --emails:          Replace ALL emails (can be repeated)
  --add-email:       Add an email without removing existing (can be repeated)
  --remove-email:    Remove an email by value (can be repeated)

Email format: "type:email" or just "email" (defaults to work)
Email types: work (default), home, other

Other fields:
  --firstname, -f: Update first name
  --lastname, -l:  Update last name
  --company, -c:   Update company name
  --position, -r:  Update role/position
  --notes, -n:     Update notes`,
		Example: `  # Update only first name
  google-contacts update c123456789 --firstname "Jane"

  # Update primary phone (backward compatible)
  google-contacts update c123456789 -p "+33698765432"

  # Replace all phones with new ones
  google-contacts update c123456789 --phones "mobile:+33612345678" --phones "work:+33123456789"

  # Add a work phone without removing existing
  google-contacts update c123456789 --add-phone "work:+33123456789"

  # Remove a specific phone
  google-contacts update c123456789 --remove-phone "+33612345678"

  # Update primary email (backward compatible)
  google-contacts update c123456789 -e "newemail@acme.com"

  # Replace all emails with new ones
  google-contacts update c123456789 --emails "work:john@acme.com" --emails "home:john@gmail.com"

  # Add a personal email without removing existing
  google-contacts update c123456789 --add-email "home:john@gmail.com"

  # Remove a specific email
  google-contacts update c123456789 --remove-email "old@acme.com"

  # Update company information
  google-contacts update c123456789 --company "New Corp" --position "CEO"`,
		Args: cobra.ExactArgs(1),
		RunE: runUpdate,
	}
)

// parsePhones parses phone strings in format "type:number" or just "number".
// Valid types: mobile (default), work, home, main, other
func parsePhones(phoneStrs []string) ([]contacts.PhoneEntry, error) {
	validTypes := map[string]bool{
		"mobile": true,
		"work":   true,
		"home":   true,
		"main":   true,
		"other":  true,
	}

	var phones []contacts.PhoneEntry
	for _, ps := range phoneStrs {
		var entry contacts.PhoneEntry
		if idx := strings.Index(ps, ":"); idx > 0 {
			// Format: type:number
			phoneType := strings.ToLower(ps[:idx])
			if !validTypes[phoneType] {
				return nil, fmt.Errorf("invalid phone type '%s', valid types: mobile, work, home, main, other", phoneType)
			}
			entry.Type = phoneType
			entry.Value = ps[idx+1:]
		} else {
			// Format: just number (defaults to mobile)
			entry.Type = "mobile"
			entry.Value = ps
		}
		if entry.Value == "" {
			return nil, fmt.Errorf("phone number cannot be empty")
		}
		phones = append(phones, entry)
	}
	return phones, nil
}

// parseEmails parses email strings in format "type:email" or just "email".
// Valid types: work (default), home, other
func parseEmails(emailStrs []string) ([]contacts.EmailEntry, error) {
	validTypes := map[string]bool{
		"work":  true,
		"home":  true,
		"other": true,
	}

	var emails []contacts.EmailEntry
	for _, es := range emailStrs {
		var entry contacts.EmailEntry
		if idx := strings.Index(es, ":"); idx > 0 {
			// Format: type:email
			emailType := strings.ToLower(es[:idx])
			if !validTypes[emailType] {
				return nil, fmt.Errorf("invalid email type '%s', valid types: work, home, other", emailType)
			}
			entry.Type = emailType
			entry.Value = es[idx+1:]
		} else {
			// Format: just email (defaults to work)
			entry.Type = "work"
			entry.Value = es
		}
		if entry.Value == "" {
			return nil, fmt.Errorf("email address cannot be empty")
		}
		emails = append(emails, entry)
	}
	return emails, nil
}

func runCreate(cmd *cobra.Command, args []string) error {
	// Validate required fields
	if createFirstName == "" {
		return fmt.Errorf("first name is required (--firstname or -f)")
	}
	if createLastName == "" {
		return fmt.Errorf("last name is required (--lastname or -l)")
	}
	if len(createPhones) == 0 {
		return fmt.Errorf("at least one phone number is required (--phone or -p)")
	}

	// Parse phone numbers
	phones, err := parsePhones(createPhones)
	if err != nil {
		return fmt.Errorf("invalid phone format: %w", err)
	}

	// Parse email addresses (optional)
	var emails []contacts.EmailEntry
	if len(createEmails) > 0 {
		emails, err = parseEmails(createEmails)
		if err != nil {
			return fmt.Errorf("invalid email format: %w", err)
		}
	}

	ctx := context.Background()

	// Get People API service
	srv, err := contacts.GetPeopleService(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize service: %w", err)
	}

	// Create the contact
	input := contacts.ContactInput{
		FirstName: createFirstName,
		LastName:  createLastName,
		Phones:    phones,
		Emails:    emails,
		Company:   createCompany,
		Position:  createPosition,
		Notes:     createNotes,
	}

	created, err := srv.CreateContact(ctx, input)
	if err != nil {
		return err
	}

	// Display success message
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Println(green("Contact created successfully!"))
	fmt.Println()
	fmt.Printf("  %s: %s\n", cyan("Name"), created.DisplayName)
	fmt.Printf("  %s: %s\n", cyan("ID"), created.ResourceName)

	return nil
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]
	ctx := context.Background()

	// Get People API service
	srv, err := contacts.GetPeopleService(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize service: %w", err)
	}

	// Search for contacts
	results, err := srv.SearchContacts(ctx, query)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Printf("No contacts found matching \"%s\"\n", query)
		return nil
	}

	// Single result: show full details
	if len(results) == 1 {
		displayContactDetails(&results[0])
		return nil
	}

	// Multiple results: show summary table
	displayContactTable(results)
	return nil
}

func runShow(cmd *cobra.Command, args []string) error {
	contactID := args[0]
	ctx := context.Background()

	// Get People API service
	srv, err := contacts.GetPeopleService(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize service: %w", err)
	}

	// Get contact details
	details, err := srv.GetContactDetails(ctx, contactID)
	if err != nil {
		return err
	}

	// Display full contact details
	displayFullContactDetails(details)
	return nil
}

func runDelete(cmd *cobra.Command, args []string) error {
	contactID := args[0]
	ctx := context.Background()

	// Get People API service
	srv, err := contacts.GetPeopleService(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize service: %w", err)
	}

	// Get contact details first (for display and confirmation)
	details, err := srv.GetContactDetails(ctx, contactID)
	if err != nil {
		return err
	}

	// Display contact summary
	displayDeleteSummary(details)

	// If not forced, ask for confirmation
	if !deleteForce {
		fmt.Print("\nAre you sure you want to delete this contact? (y/N): ")
		var response string
		fmt.Scanln(&response)

		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	// Delete the contact
	err = srv.DeleteContact(ctx, contactID)
	if err != nil {
		return err
	}

	// Display success message
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Println()
	fmt.Printf("%s Contact '%s' has been deleted.\n", green("✓"), details.DisplayName)

	return nil
}

func runUpdate(cmd *cobra.Command, args []string) error {
	contactID := args[0]
	ctx := context.Background()

	// Build update input - only set fields that were explicitly provided
	// Check this BEFORE making API calls to fail fast
	input := contacts.UpdateInput{}
	hasUpdates := false

	if cmd.Flags().Changed("firstname") {
		input.FirstName = &updateFirstName
		hasUpdates = true
	}
	if cmd.Flags().Changed("lastname") {
		input.LastName = &updateLastName
		hasUpdates = true
	}

	// Phone update options (in priority order)
	if cmd.Flags().Changed("phone") {
		// Backward compatible: replaces first phone
		input.Phone = &updatePhone
		hasUpdates = true
	}
	if len(updatePhones) > 0 {
		// Replaces all phones
		phones, err := parsePhones(updatePhones)
		if err != nil {
			return fmt.Errorf("invalid --phones format: %w", err)
		}
		input.Phones = phones
		hasUpdates = true
	}
	if len(updateAddPhones) > 0 {
		// Add phones without removing existing
		phones, err := parsePhones(updateAddPhones)
		if err != nil {
			return fmt.Errorf("invalid --add-phone format: %w", err)
		}
		input.AddPhones = phones
		hasUpdates = true
	}
	if len(updateRemPhones) > 0 {
		// Remove phones by value
		input.RemovePhones = updateRemPhones
		hasUpdates = true
	}

	// Email update options (in priority order)
	if cmd.Flags().Changed("email") {
		// Backward compatible: replaces first email
		input.Email = &updateEmail
		hasUpdates = true
	}
	if len(updateEmails) > 0 {
		// Replaces all emails
		emails, err := parseEmails(updateEmails)
		if err != nil {
			return fmt.Errorf("invalid --emails format: %w", err)
		}
		input.Emails = emails
		hasUpdates = true
	}
	if len(updateAddEmails) > 0 {
		// Add emails without removing existing
		emails, err := parseEmails(updateAddEmails)
		if err != nil {
			return fmt.Errorf("invalid --add-email format: %w", err)
		}
		input.AddEmails = emails
		hasUpdates = true
	}
	if len(updateRemEmails) > 0 {
		// Remove emails by value
		input.RemoveEmails = updateRemEmails
		hasUpdates = true
	}

	if cmd.Flags().Changed("company") {
		input.Company = &updateCompany
		hasUpdates = true
	}
	if cmd.Flags().Changed("position") {
		input.Position = &updatePosition
		hasUpdates = true
	}
	if cmd.Flags().Changed("notes") {
		input.Notes = &updateNotes
		hasUpdates = true
	}

	// Check if any fields were provided
	if !hasUpdates {
		return fmt.Errorf("no fields specified to update. Use --help to see available flags")
	}

	// Get People API service
	srv, err := contacts.GetPeopleService(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize service: %w", err)
	}

	// Get current contact details first (for before display)
	beforeDetails, err := srv.GetContactDetails(ctx, contactID)
	if err != nil {
		return err
	}

	// Perform the update
	afterDetails, err := srv.UpdateContact(ctx, contactID, input)
	if err != nil {
		return err
	}

	// Display success message with before/after summary
	displayUpdateSummary(beforeDetails, afterDetails)

	return nil
}

// displayUpdateSummary shows the before/after contact details.
func displayUpdateSummary(before, after *contacts.ContactDetails) {
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	fmt.Println(green("Contact updated successfully!"))
	fmt.Println()

	// Name
	if before.DisplayName != after.DisplayName {
		fmt.Printf("  %s: %s → %s\n", cyan("Name"), yellow(before.DisplayName), green(after.DisplayName))
	} else {
		fmt.Printf("  %s: %s\n", cyan("Name"), after.DisplayName)
	}

	fmt.Printf("  %s: %s\n", cyan("ID"), extractID(after.ResourceName))

	// Phone (compare first phone)
	beforePhone := ""
	afterPhone := ""
	if len(before.Phones) > 0 {
		beforePhone = before.Phones[0].Value
	}
	if len(after.Phones) > 0 {
		afterPhone = after.Phones[0].Value
	}
	if beforePhone != afterPhone && afterPhone != "" {
		fmt.Printf("  %s: %s → %s\n", cyan("Phone"), yellow(beforePhone), green(afterPhone))
	} else if afterPhone != "" {
		fmt.Printf("  %s: %s\n", cyan("Phone"), afterPhone)
	}

	// Email (compare first email)
	beforeEmail := ""
	afterEmail := ""
	if len(before.Emails) > 0 {
		beforeEmail = before.Emails[0].Value
	}
	if len(after.Emails) > 0 {
		afterEmail = after.Emails[0].Value
	}
	if beforeEmail != afterEmail && afterEmail != "" {
		fmt.Printf("  %s: %s → %s\n", cyan("Email"), yellow(beforeEmail), green(afterEmail))
	} else if afterEmail != "" {
		fmt.Printf("  %s: %s\n", cyan("Email"), afterEmail)
	}

	// Company
	if before.Company != after.Company && after.Company != "" {
		fmt.Printf("  %s: %s → %s\n", cyan("Company"), yellow(before.Company), green(after.Company))
	} else if after.Company != "" {
		fmt.Printf("  %s: %s\n", cyan("Company"), after.Company)
	}

	// Position
	if before.Position != after.Position && after.Position != "" {
		fmt.Printf("  %s: %s → %s\n", cyan("Position"), yellow(before.Position), green(after.Position))
	} else if after.Position != "" {
		fmt.Printf("  %s: %s\n", cyan("Position"), after.Position)
	}

	// Notes
	if before.Notes != after.Notes && after.Notes != "" {
		fmt.Printf("  %s: (updated)\n", cyan("Notes"))
	} else if after.Notes != "" {
		fmt.Printf("  %s: %s\n", cyan("Notes"), truncate(after.Notes, 50))
	}
}

// displayDeleteSummary shows contact summary before deletion.
func displayDeleteSummary(details *contacts.ContactDetails) {
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	fmt.Println(yellow("Contact to delete:"))
	fmt.Println()
	fmt.Printf("  %s: %s\n", cyan("Name"), details.DisplayName)
	fmt.Printf("  %s: %s\n", cyan("ID"), extractID(details.ResourceName))

	if len(details.Phones) > 0 {
		fmt.Printf("  %s: %s\n", cyan("Phone"), details.Phones[0].Value)
	}
	if len(details.Emails) > 0 {
		fmt.Printf("  %s: %s\n", cyan("Email"), details.Emails[0].Value)
	}
	if details.Company != "" {
		fmt.Printf("  %s: %s\n", cyan("Company"), details.Company)
	}
}

// displayContactDetails shows full information for a single contact (from search result).
func displayContactDetails(contact *contacts.SearchResult) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Println(green("Contact found:"))
	fmt.Println()
	fmt.Printf("  %s: %s\n", cyan("Name"), contact.DisplayName)
	fmt.Printf("  %s: %s\n", cyan("ID"), extractID(contact.ResourceName))

	if contact.Phone != "" {
		fmt.Printf("  %s: %s\n", cyan("Phone"), contact.Phone)
	}
	if contact.Email != "" {
		fmt.Printf("  %s: %s\n", cyan("Email"), contact.Email)
	}
	if contact.Company != "" {
		fmt.Printf("  %s: %s\n", cyan("Company"), contact.Company)
	}
	if contact.Position != "" {
		fmt.Printf("  %s: %s\n", cyan("Position"), contact.Position)
	}
	if contact.Notes != "" {
		fmt.Printf("  %s: %s\n", cyan("Notes"), contact.Notes)
	}
}

// displayFullContactDetails shows complete information for a contact (from show command).
func displayFullContactDetails(details *contacts.ContactDetails) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	fmt.Println(green("Contact Details"))
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println()

	// Name section
	fmt.Printf("  %s: %s\n", cyan("Name"), details.DisplayName)
	if details.FirstName != "" || details.LastName != "" {
		fmt.Printf("    %s: %s\n", yellow("First"), details.FirstName)
		fmt.Printf("    %s: %s\n", yellow("Last"), details.LastName)
	}

	// Contact ID
	fmt.Printf("  %s: %s\n", cyan("ID"), extractID(details.ResourceName))
	fmt.Println()

	// Phone numbers
	if len(details.Phones) > 0 {
		if len(details.Phones) == 1 {
			fmt.Printf("  %s: %s (%s)\n", cyan("Phone"), details.Phones[0].Value, yellow(details.Phones[0].Type))
		} else {
			fmt.Printf("  %s:\n", cyan("Phones"))
			for _, phone := range details.Phones {
				fmt.Printf("    • %s (%s)\n", phone.Value, yellow(phone.Type))
			}
		}
	}

	// Email addresses
	if len(details.Emails) > 0 {
		if len(details.Emails) == 1 {
			fmt.Printf("  %s: %s (%s)\n", cyan("Email"), details.Emails[0].Value, yellow(details.Emails[0].Type))
		} else {
			fmt.Printf("  %s:\n", cyan("Emails"))
			for _, email := range details.Emails {
				fmt.Printf("    • %s (%s)\n", email.Value, yellow(email.Type))
			}
		}
	}

	// Organization
	if details.Company != "" || details.Position != "" {
		fmt.Println()
		if details.Company != "" {
			fmt.Printf("  %s: %s\n", cyan("Company"), details.Company)
		}
		if details.Position != "" {
			fmt.Printf("  %s: %s\n", cyan("Position"), details.Position)
		}
	}

	// Notes
	if details.Notes != "" {
		fmt.Println()
		fmt.Printf("  %s:\n", cyan("Notes"))
		// Indent multiline notes
		for _, line := range strings.Split(details.Notes, "\n") {
			fmt.Printf("    %s\n", line)
		}
	}

	// Metadata
	if details.UpdatedAt != "" {
		fmt.Println()
		fmt.Printf("  %s: %s\n", cyan("Updated"), formatTime(details.UpdatedAt))
	}
}

// displayContactTable shows a summary table for multiple contacts.
func displayContactTable(results []contacts.SearchResult) {
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("Found %d contacts:\n\n", len(results))

	// Use tabwriter for aligned columns
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
		cyan("ID"),
		cyan("Name"),
		cyan("Phone"),
		cyan("Company"),
		cyan("Email"))

	// Separator
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
		strings.Repeat("-", 15),
		strings.Repeat("-", 20),
		strings.Repeat("-", 15),
		strings.Repeat("-", 15),
		strings.Repeat("-", 25))

	// Data rows
	for _, r := range results {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			extractID(r.ResourceName),
			truncate(r.DisplayName, 20),
			truncate(r.Phone, 15),
			truncate(r.Company, 15),
			truncate(r.Email, 25))
	}
}

// extractID extracts the contact ID from a resource name.
func extractID(resourceName string) string {
	if len(resourceName) > 7 && resourceName[:7] == "people/" {
		return resourceName[7:]
	}
	return resourceName
}

// truncate shortens a string to the specified length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// formatTime formats an ISO 8601 timestamp for display.
func formatTime(isoTime string) string {
	// Input format from Google API: "2026-01-14T10:30:00.123456Z"
	// Try to parse and format nicely
	if len(isoTime) >= 10 {
		// Extract date and time parts
		date := isoTime[:10] // "2026-01-14"
		if len(isoTime) >= 19 {
			time := isoTime[11:19] // "10:30:00"
			return fmt.Sprintf("%s %s", date, time)
		}
		return date
	}
	return isoTime
}

// Init initializes the CLI commands and flags.
func Init() {
	// Add version flag to root command
	RootCmd.Version = Version
	RootCmd.SetVersionTemplate("google-contacts version {{.Version}}\n")

	// Setup create command flags
	createCmd.Flags().StringVarP(&createFirstName, "firstname", "f", "", "First name (required)")
	createCmd.Flags().StringVarP(&createLastName, "lastname", "l", "", "Last name (required)")
	createCmd.Flags().StringArrayVarP(&createPhones, "phone", "p", nil, "Phone number (can be repeated, format: 'type:number' or 'number')")
	createCmd.Flags().StringArrayVarP(&createEmails, "email", "e", nil, "Email address (can be repeated, format: 'type:email' or 'email')")
	createCmd.Flags().StringVarP(&createCompany, "company", "c", "", "Company name")
	createCmd.Flags().StringVarP(&createPosition, "position", "r", "", "Role/position at company")
	createCmd.Flags().StringVarP(&createNotes, "notes", "n", "", "Notes about the contact")

	// Setup delete command flags
	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Skip confirmation prompt")

	// Setup update command flags
	updateCmd.Flags().StringVarP(&updateFirstName, "firstname", "f", "", "First name")
	updateCmd.Flags().StringVarP(&updateLastName, "lastname", "l", "", "Last name")
	updateCmd.Flags().StringVarP(&updatePhone, "phone", "p", "", "Primary phone (replaces first phone)")
	updateCmd.Flags().StringArrayVar(&updatePhones, "phones", nil, "Replace ALL phones (can be repeated, format: 'type:number')")
	updateCmd.Flags().StringArrayVar(&updateAddPhones, "add-phone", nil, "Add phone without removing existing (can be repeated)")
	updateCmd.Flags().StringArrayVar(&updateRemPhones, "remove-phone", nil, "Remove phone by value (can be repeated)")
	updateCmd.Flags().StringVarP(&updateEmail, "email", "e", "", "Primary email (replaces first email)")
	updateCmd.Flags().StringArrayVar(&updateEmails, "emails", nil, "Replace ALL emails (can be repeated, format: 'type:email')")
	updateCmd.Flags().StringArrayVar(&updateAddEmails, "add-email", nil, "Add email without removing existing (can be repeated)")
	updateCmd.Flags().StringArrayVar(&updateRemEmails, "remove-email", nil, "Remove email by value (can be repeated)")
	updateCmd.Flags().StringVarP(&updateCompany, "company", "c", "", "Company name")
	updateCmd.Flags().StringVarP(&updatePosition, "position", "r", "", "Role/position at company")
	updateCmd.Flags().StringVarP(&updateNotes, "notes", "n", "", "Notes about the contact")

	// Register commands
	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(createCmd)
	RootCmd.AddCommand(searchCmd)
	RootCmd.AddCommand(showCmd)
	RootCmd.AddCommand(deleteCmd)
	RootCmd.AddCommand(updateCmd)
}
