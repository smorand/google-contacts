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
	createPhone     string
	createEmail     string
	createCompany   string
	createPosition  string
	createNotes     string
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
  --phone, -p:     Phone number

Recommended fields:
  --company, -c:   Company name

Optional fields:
  --email, -e:     Email address
  --position, -r:  Role/position at company
  --notes, -n:     Notes about the contact`,
		Example: `  # Create contact with required fields only
  google-contacts create -f John -l Doe -p +33612345678

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
)

func runCreate(cmd *cobra.Command, args []string) error {
	// Validate required fields
	if createFirstName == "" {
		return fmt.Errorf("first name is required (--firstname or -f)")
	}
	if createLastName == "" {
		return fmt.Errorf("last name is required (--lastname or -l)")
	}
	if createPhone == "" {
		return fmt.Errorf("phone number is required (--phone or -p)")
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
		Phone:     createPhone,
		Email:     createEmail,
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
	createCmd.Flags().StringVarP(&createPhone, "phone", "p", "", "Phone number (required)")
	createCmd.Flags().StringVarP(&createEmail, "email", "e", "", "Email address")
	createCmd.Flags().StringVarP(&createCompany, "company", "c", "", "Company name")
	createCmd.Flags().StringVarP(&createPosition, "position", "r", "", "Role/position at company")
	createCmd.Flags().StringVarP(&createNotes, "notes", "n", "", "Notes about the contact")

	// Register commands
	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(createCmd)
	RootCmd.AddCommand(searchCmd)
	RootCmd.AddCommand(showCmd)
}
