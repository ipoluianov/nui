package contacts

import (
	"strings"
	"time"
)

// Contact is one entry of the address book.
type Contact struct {
	Name     string
	Company  string
	Group    string
	Phone    string
	Email    string
	Birthday time.Time
	Favorite bool
	Notes    string
}

// groups are the groups a contact can be in.
var groups = []string{"Family", "Friends", "Work", "Other"}

// matches reports whether the contact contains the search text in any of
// its visible fields; text must be lower case.
func (c *Contact) matches(text string) bool {
	for _, field := range []string{c.Name, c.Company, c.Phone, c.Email} {
		if strings.Contains(strings.ToLower(field), text) {
			return true
		}
	}
	return false
}

// age is the contact's age in whole years at the given moment.
func (c *Contact) age(now time.Time) int {
	years := now.Year() - c.Birthday.Year()
	if now.YearDay() < c.Birthday.YearDay() {
		years--
	}
	return years
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.Local)
}

func sampleContacts() []*Contact {
	return []*Contact{
		{"Alice Johnson", "Northwind Traders", "Work", "+1 555 0101", "alice.johnson@northwind.example", date(1988, time.March, 14), true, "Project lead for the Q4 rollout.\nPrefers calls before noon."},
		{"Bob Smith", "", "Friends", "+1 555 0102", "bob.smith@mail.example", date(1990, time.July, 2), false, "Plays bass in a garage band."},
		{"Carol White", "Contoso Ltd", "Work", "+1 555 0103", "carol@contoso.example", date(1979, time.November, 23), false, "Accounting. Invoices go to her."},
		{"David Brown", "", "Family", "+1 555 0104", "david.brown@mail.example", date(1962, time.January, 8), true, "Uncle David. Birthday dinner every January."},
		{"Emma Davis", "Fabrikam", "Work", "+1 555 0105", "emma.davis@fabrikam.example", date(1993, time.May, 30), false, ""},
		{"Frank Miller", "", "Friends", "+1 555 0106", "frank.m@mail.example", date(1985, time.September, 17), false, "Met at the climbing gym."},
		{"Grace Wilson", "Adventure Works", "Work", "+44 20 7946 0107", "grace.wilson@adventure.example", date(1991, time.February, 11), false, "London office, GMT."},
		{"Henry Moore", "", "Family", "+1 555 0108", "henry.moore@mail.example", date(2001, time.June, 5), false, "Cousin, studies architecture."},
		{"Isabel Taylor", "Tailspin Toys", "Other", "+1 555 0109", "isabel@tailspin.example", date(1983, time.December, 1), false, "Supplier of the office coffee machine."},
		{"Jack Anderson", "", "Friends", "+1 555 0110", "jack.anderson@mail.example", date(1987, time.August, 21), true, "Best man at the wedding."},
		{"Karen Thomas", "Litware Inc", "Work", "+1 555 0111", "karen.thomas@litware.example", date(1975, time.April, 3), false, "CTO. Formal emails only."},
		{"Liam Jackson", "", "Other", "+1 555 0112", "liam.j@mail.example", date(1995, time.October, 27), false, "Plumber, reliable, works weekends."},
		{"Mia Harris", "", "Family", "+1 555 0113", "mia.harris@mail.example", date(1998, time.March, 9), false, "Sister-in-law."},
		{"Noah Martin", "Woodgrove Bank", "Work", "+1 555 0114", "noah.martin@woodgrove.example", date(1982, time.July, 19), false, "Mortgage advisor."},
		{"Olivia Garcia", "", "Friends", "+34 91 555 0115", "olivia.garcia@mail.example", date(1992, time.January, 25), false, "Lives in Madrid now."},
	}
}
