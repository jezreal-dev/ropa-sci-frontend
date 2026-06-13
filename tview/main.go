package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	// "strings"
	"golang.org/x/crypto/bcrypt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// --- Models & Globals ---

var states = []string{"AB", "AD", "AK", "AN", "BA", "BY", "BE", "BO", "CR",
	"DE", "EB", "ED", "EK", "EN", "FC", "GO", "IM", "JI", "KD", "KN", "KT",
	"KE", "KO", "KW", "LA", "NA", "NI", "OG", "ON", "OS", "OY", "PL", "RI", "SO", "TA", "YO", "ZA",
}

// ------Contac
type Contact struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	UserName	string `json:"UserName"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
	Gender      string `json:"gender"`
	State       string `json:"state"`
	PassWord	string `json:"Password"`

}

var contacts []Contact
var app = tview.NewApplication()
var pages = tview.NewPages()

// Layout Components
var flex = tview.NewFlex()
var contactText = tview.NewTextView().SetDynamicColors(true)
var contactsList = tview.NewList().ShowSecondaryText(false)
var form = tview.NewForm()
var profileFlex = tview.NewFlex()
var profileText = tview.NewTextView().SetDynamicColors(true)
var welcomeFlex = tview.NewFlex()
var welcomeText = tview.NewTextView().SetDynamicColors(true)
var footerText = tview.NewTextView().
	SetTextColor(tcell.ColorGreen).
	SetText("(q) to quit | (m) menu")
	// SetText("(a) to add a new contact | (q) to quit | (m) menu")

// --- Main Execution ---

func main() {
	// 1. Load data
	if err := loadContacts(); err != nil {
		fmt.Printf("Error loading contacts: %v\n", err)
	}

	// 2. Initialize UI Components
	addContactList()
	welcomePage() // Build the welcome screen logic

	// 3. Main Menu Navigation Logic
	contactsList.SetSelectedFunc(func(index int, name string, second_name string, shortcut rune) {
		if index < len(contacts) {
			showProfile(&contacts[index])
			pages.SwitchToPage("Profile")
		}
	})

	// 4. Setup Main Menu Layout
	flex.SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(contactsList, 0, 1, true).
			AddItem(contactText, 0, 2, false), 0, 6, true).
		AddItem(footerText, 1, 1, false)

	// 5. Global Keyboard Shortcuts
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		frontPage, _ := pages.GetFrontPage()

		if frontPage == "Add Contact" || frontPage == "Signin Page" || frontPage == "GameInput" {
			return event
		}

		if event.Rune() == 'q' {
			app.Stop()
		} else if event.Rune() == 'm' {
			pages.SwitchToPage("Welcome Page")
		}
		// else if event.Rune() == 'a' {
		// 	addContactForm()
		// 	pages.SwitchToPage("Add Contact")
		
		return event
	})

	// 6. Define Pages
	// Pages: Name, Item, Resize, Visible
	pages.AddPage("Welcome Page", welcomeFlex, true, true) 
	pages.AddPage("Menu", flex, true, false)
	pages.AddPage("Profile", profileFlex, true, false)
	pages.AddPage("Add Contact", form, true, false)
	pages.AddPage("Signin Page", form, true, false)

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

// --- UI Building Functions ---

func welcomePage() {
	welcomeFlex.Clear()
	welcomeText.Clear()

	// Generate ASCII art using the tview-compatible color function
	// Note: Keep this string short to prevent wrapping on standard terminals
	ascii := asciiColor("WELCOME TO GAME ARENA", "", "red")
	welcomeText.SetText(ascii.String()).SetTextAlign(tview.AlignCenter)

	welcomeButtons := tview.NewForm().
		// AddButton("Enter Application", func() {
		// 	pages.SwitchToPage("Menu")
		// }).
		AddButton("Sign in", func() {
			// addContactForm()
			login()
			pages.SwitchToPage("Signin Page")
		}).
		AddButton("Register", func ()  {
			addContactForm()
			pages.SwitchToPage("Add Contact")
		}).
		AddButton("Quit", func() {
			app.Stop()
		})
	welcomeButtons.SetButtonsAlign(tview.AlignCenter)

	welcomeFlex.SetDirection(tview.FlexRow).
		AddItem(tview.NewBox(), 0, 1, false).      // Top Spacer
		AddItem(welcomeText, 10, 1, false).         // ASCII Art (approx 8-10 lines high)
		AddItem(welcomeButtons, 0, 1, true).       // Buttons (focused)
		AddItem(tview.NewBox(), 0, 1, false)       // Bottom Spacer
}

func login() {
    form.Clear(true)
    
    var enteredUsername string
    var enteredPassword string

    // 1. Capture the input into local variables
    form.AddInputField("Username", "", 20, nil, func(text string) {
        enteredUsername = text
    })
    form.AddPasswordField("Password", "", 20, '*', func(text string) {
        enteredPassword = text
    })

	// form.AddInputField("Password", "", 20, nil, func(text string) {
    //     enteredPassword = text
    // })

    form.AddButton("Login", func() {
        success := false
        var matchedContact *Contact

        // 2. Loop through the contacts list to find the user
        for i := range contacts {
            // Here I'm assuming 'FirstName' is the username. 
            // You could also use Email!
            if contacts[i].FirstName == enteredUsername {
                // 3. Compare the hashed password
                // Note: You need to have a 'Password' field in your Contact struct
                err := bcrypt.CompareHashAndPassword([]byte(contacts[i].PassWord), []byte(enteredPassword))
                
                if err == nil {
                    success = true
                    matchedContact = &contacts[i]
                    break

                }
            }
        }

        if success {
            showProfile(matchedContact)
            pages.SwitchToPage("Profile")
        } else {
            // Optional: You could update a text view to say "Invalid Login"
            // For now, let's just clear the password field
            form.GetFormItemByLabel("Password").(*tview.InputField).SetText("")
        }
    })

    form.AddButton("Cancel", func() {
        pages.SwitchToPage("Menu")
    })
}

func showProfile(contact *Contact) {
	profileFlex.Clear()
	profileText.Clear()

	details := fmt.Sprintf(
		"\n [yellow]CONTACT PROFILE[white]\n--------------------------\n\n"+
			" [green]Name:[white]   %s %s\n"+
			" [green]Email:[white]  %s\n"+
			" [green]Phone:[white]  %s\n"+
			" [green]State:[white]  %s\n"+
			" [green]Gender:[white] %s\n",
		contact.FirstName, contact.LastName, contact.Email, contact.PhoneNumber, contact.State, contact.Gender,
	)
	profileText.SetText(details).SetTextAlign(tview.AlignCenter)

	profileButtons := tview.NewForm().
		// AddButton("Back to Menu", func() {
		// 	pages.SwitchToPage("Menu")
		// }).
		AddButton("Play Game", func ()  {
			startBattle(contact)
			// pages.SwitchToPage("Game")
		}).
		AddButton("See Avaliable Players", func ()  {
			pages.SwitchToPage("Menu")
		})
	profileButtons.SetButtonsAlign(tview.AlignCenter)

	profileFlex.SetDirection(tview.FlexRow).
		AddItem(profileText, 0, 1, false).
		AddItem(profileButtons, 3, 1, true)
}

func addContactForm() {
	form.Clear(true)
	contact := &Contact{}

	form.AddInputField("First Name", "", 20, nil, func(text string) { contact.FirstName = text })
	form.AddInputField("Last Name", "", 20, nil, func(text string) { contact.LastName = text })
	form.AddInputField("UserName", "", 20, nil, func(text string) { contact.UserName = text })
	form.AddInputField("Email", "", 20, nil, func(text string) { contact.Email = text })
	form.AddInputField("Phone Number", "", 20, nil, func(text string) { contact.PhoneNumber = text })
	form.AddInputField("Gender", "", 20, nil, func(text string) { contact.Gender = text })
	form.AddDropDown("State", states, 0, func(option string, index int) { contact.State = option })
	form.AddPasswordField("PassWord", "", 20, '*', func(text string) { contact.PassWord = hashPassword(text) })

	form.AddButton("Submit", func() {
		contacts = append(contacts, *contact)
		saveContacts()
		// addContactList()
		login()
		pages.SwitchToPage("Signin Page")
	})
	form.AddButton("Cancel", func() {
		pages.SwitchToPage("Menu")
	})
}

func addContactList() {
	contactsList.Clear()
	for index, contact := range contacts {
		label := fmt.Sprintf("%s %s (%s)", contact.FirstName, contact.LastName, contact.State)
		contactsList.AddItem(label, "", rune('1'+index), nil)
	}
}


// saved contact to the db -json
func saveContacts() {
	b, err := json.MarshalIndent(contacts, "", " ") // #nosec G117
	if err != nil {
		return
	}
	if err := os.WriteFile("data.json", b, 0600); err != nil {
		return
	}
}

// Load contact from the json db
func loadContacts() error {
	if _, err := os.Stat("data.json"); os.IsNotExist(err) {
		return nil
	}
	data, err := os.ReadFile("data.json")
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &contacts)
}

// hash password
func hashPassword(password string) string  {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	
	if err != nil {
		panic(err)
	}
	return string(hash)
}

func startBattle(c1 *Contact) {
    battleForm := tview.NewForm()
    var user1Choice string

    // 1. Setup the user's choice dropdown
    battleForm.AddDropDown("Your Tool", []string{"rock", "paper", "scissors"}, 0, func(o string, i int) {
        user1Choice = o
    })

    // 2. The Battle Button
    battleForm.AddButton("BATTLE!", func() {
        choices := []string{"rock", "paper", "scissors"}
        opponentChoice := choices[rand.IntN(3)] // #nosec G404

        // Create the result view
        resultView := tview.NewTextView().
            SetDynamicColors(true).
            SetTextAlign(tview.AlignCenter).
            SetWrap(false) // ASCII art must not wrap

        // Get the ASCII art
        suspenseArt := RunRPSBattle(c1.FirstName, "CPU", user1Choice, opponentChoice)
        
        // Hide the winner line for the "delay" effect
        lines := strings.Split(suspenseArt, "\n")
        var initialArt string
        if len(lines) > 2 {
            initialArt = strings.Join(lines[:len(lines)-2], "\n")
        } else {
            initialArt = suspenseArt
        }

        resultView.SetText(initialArt + "\n\n[yellow]Calculating result...[white]")
        pages.AddPage("BattleResult", resultView, true, true)
        pages.SwitchToPage("BattleResult") // Show the suspense screen

        go func() {
            time.Sleep(2 * time.Second)
            app.QueueUpdateDraw(func() {
                resultView.SetText(suspenseArt + "\n\n[gray]Press Enter to go back[white]")
                
                // Allow user to exit the result screen
                resultView.SetDoneFunc(func(key tcell.Key) {
                    pages.RemovePage("BattleResult")
                    pages.RemovePage("GameInput")
                    pages.SwitchToPage("Profile")
                })
            })
        }()
    })

    // 3. IMPORTANT: Add a Cancel button
    battleForm.AddButton("Cancel", func() {
        pages.RemovePage("GameInput")
        pages.SwitchToPage("Profile")
    })

    // 4. Add and EXPLICITLY switch
    pages.AddPage("GameInput", battleForm, true, true)
    pages.SwitchToPage("GameInput") 
}

// RunRPSBattle takes the players and their choices and returns the visual result
func RunRPSBattle(user1, user2, choice1, choice2 string) string {
    // 1. Load ASCII Files (Error handling simplified for brevity)
    gesture, _ := os.ReadFile("ropa-sci.txt")
    revGestureFile, _ := os.Open("ropa-sci-mirror.txt")
    defer revGestureFile.Close()

    // Parse gestures into slices (using your logic)
    var gestureLines []string
    begin := 0
    for i := range gesture {
        if gesture[i] == '\n' {
            gestureLines = append(gestureLines, string(gesture[begin:i]))
            begin = i + 1
        }
    }

    var revGestureLines []string
    revScanner := bufio.NewScanner(revGestureFile)
    for revScanner.Scan() {
        revGestureLines = append(revGestureLines, revScanner.Text())
    }

    // 2. Map Choices to ASCII
    options := make(map[string][]string)
    revOptions := make(map[string][]string)
    list := []string{"rock", "paper", "scissors"}
    height := 7

    for i := range list {
        length := i * height
        options[list[i]] = gestureLines[length : length+height]
        revOptions[list[i]] = revGestureLines[length : length+height]
    }

    // 3. Build the Battle Scene String
    var battleScene strings.Builder
    battleScene.WriteString(fmt.Sprintf("\n[yellow]%s (%s) vs %s (%s)[white]\n\n", user1, choice1, user2, choice2))

    for i := 0; i < height; i++ {
        // Add User 1's gesture
        battleScene.WriteString(options[choice1][i])
        // Add some spacing between them
        battleScene.WriteString("       ")
        // Add User 2's gesture
        battleScene.WriteString(revOptions[choice2][i] + "\n")
    }

    // 4. Determine Winner
    var result string
    if choice1 == choice2 {
        result = "DRAW"
    } else if (choice1 == "scissors" && choice2 == "paper") ||
        (choice1 == "paper" && choice2 == "rock") ||
        (choice1 == "rock" && choice2 == "scissors") {
        result = user1 + " WINS!"
    } else {
        result = user2 + " WINS!"
    }

    battleScene.WriteString("\n\n[green]" + result + "[white]")
    return battleScene.String()
}