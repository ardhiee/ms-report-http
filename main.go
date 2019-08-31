package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"ms-report-http/util"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jung-kurt/gofpdf"
)

type activity struct {
	Timestamp  string
	UserID     string
	Activities string
}

// Query the database
func selectactivities(userID string) []activity {
	var result []activity

	// connect to db
	db, err := connectdb()
	if err != nil {
		fmt.Println("Can't connect DB")
		fmt.Println(err.Error())
	}
	defer db.Close()

	// query the activities
	var userid = userID
	rows, err := db.Query("select timestamp, userid, activities from tb_report where userid like ? order by timestamp desc", userid)
	if err != nil {
		fmt.Println("Can't query the table")
		fmt.Println(err.Error())
	}
	defer db.Close()

	for rows.Next() {
		var each = activity{}
		var err = rows.Scan(&each.Timestamp, &each.UserID, &each.Activities)

		if err != nil {
			fmt.Println(err.Error())
		}

		result = append(result, each)

	}
	return result
}

// JSON WEB API
func useractivities(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	var id = r.FormValue("userid")
	if id == "" {
		id = "%"
	}
	var data = selectactivities(id)

	if r.Method == "GET" {
		var result, err = json.Marshal(data)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(result)
		return
	}
	http.Error(w, "", http.StatusBadRequest)

}

// JSON WEB API
func generatexls(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// GenerateExcel()
	var id = r.FormValue("userid")
	if id == "" {
		id = "%"
	}

	var filepath = GenerateExcel(id)

	if r.Method == "GET" {
		var result, err = json.Marshal(filepath)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(result)
		return
	}
	http.Error(w, "", http.StatusBadRequest)

}

func generatepdf(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// GeneratePDF()
	var id = r.FormValue("userid")
	if id == "" {
		id = "%"
	}

	var filepath = GeneratePDF(id)

	if r.Method == "GET" {
		var result, err = json.Marshal(filepath)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(result)
		return
	}
	http.Error(w, "", http.StatusBadRequest)

}

// Handle URL http://127.0.0.1/?file=IWantToDownloadThisFile.zip
func fileurl(w http.ResponseWriter, r *http.Request) {
	//First of check if Get is set in the URL
	var Filename = r.FormValue("file")

	if Filename == "" {
		//Get not set, send a 400 bad request
		http.Error(w, "Get 'file' not specified in url.", 400)
		return
	}

	// fmt.Println("Client requests: " + Filename)

	//Check if file exists and open
	Openfile, err := os.Open(Filename)
	defer Openfile.Close() //Close after function return
	if err != nil {
		//File not found, send 404
		http.Error(w, "File not found.", 404)
		return
	}

	//File is found, create and send the correct headers

	//Get the Content-Type of the file
	//Create a buffer to store the header of the file in
	FileHeader := make([]byte, 512)
	//Copy the headers into the FileHeader buffer
	Openfile.Read(FileHeader)
	//Get content type of file
	FileContentType := http.DetectContentType(FileHeader)

	//Get the file size
	FileStat, _ := Openfile.Stat()                     //Get info from file
	FileSize := strconv.FormatInt(FileStat.Size(), 10) //Get file size as a string

	//Send the headers
	w.Header().Set("Content-Disposition", "attachment; filename="+Filename)
	w.Header().Set("Content-Type", FileContentType)
	w.Header().Set("Content-Length", FileSize)

	//Send the file
	//We read 512 bytes from the file already, so we reset the offset back to 0
	Openfile.Seek(0, 0)
	io.Copy(w, Openfile) //'Copy' the file to the client
	return
}

// HTTP Server
func main() {

	http.HandleFunc("/ms-report/useractivities", useractivities)
	http.HandleFunc("/ms-report/generatexls", generatexls)
	http.HandleFunc("/ms-report/fileurl", fileurl)
	http.HandleFunc("/ms-report/generatepdf", generatepdf)

	fmt.Println("Starting Web Server at http://localhost:8888/")
	http.ListenAndServe(":8888", nil)
}

// connect to database
func connectdb() (*sql.DB, error) {
	db, err := sql.Open("mysql", "root:password@tcp("+os.Getenv("DBCONN")+")/poc")
	if err != nil {
		fmt.Println("Fail 1")
		panic("dbpool init >> " + err.Error())
	}

	return db, nil
}

// GenerateExcel for activities in form of xlsx
func GenerateExcel(UserID string) (filepath string) {
	// M type with interface, after that initialize data excelreportdata
	type M map[string]interface{}
	var excelreportdata = []M{}
	var id = UserID

	xlsx := excelize.NewFile()
	var activities = selectactivities(id)

	// convert the string Array from DB into Map type
	for _, activity := range activities {
		var temp = M{}
		temp["Timestamp"] = activity.Timestamp
		temp["UserID"] = activity.UserID
		temp["Activities"] = activity.Activities
		excelreportdata = append(excelreportdata, temp)
	}

	sheet1Name := "Sheet One"
	xlsx.SetSheetName(xlsx.GetSheetName(1), sheet1Name)

	xlsx.SetCellValue(sheet1Name, "A1", "Timestamp")
	xlsx.SetCellValue(sheet1Name, "B1", "UserID")
	xlsx.SetCellValue(sheet1Name, "C1", "Activities")

	err := xlsx.AutoFilter(sheet1Name, "A1", "C1", "")
	if err != nil {
		log.Fatal("ERROR", err.Error())
	}

	for i, each := range excelreportdata {
		xlsx.SetCellValue(sheet1Name, fmt.Sprintf("A%d", i+2), each["Timestamp"])
		xlsx.SetCellValue(sheet1Name, fmt.Sprintf("B%d", i+2), each["UserID"])
		xlsx.SetCellValue(sheet1Name, fmt.Sprintf("C%d", i+2), each["Activities"])
	}

	t := time.Now().Format("20060102-150405")
	filepath = "./Activities-Report-" + t + ".xlsx"

	err = xlsx.SaveAs(filepath)
	if err != nil {
		fmt.Println(err)
	}

	return filepath
}

// GeneratePDF for activities in form of pdf
func GeneratePDF(UserID string) (filepath string) {

	var id = UserID
	var activities = selectactivities(id)

	marginCell := 2. // margin of top/bottom of cell
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Arial", "", 12)
	pdf.AddPage()
	pagew, pageh := pdf.GetPageSize()
	mleft, mright, _, mbottom := pdf.GetMargins()

	cols := []float64{100, 60, pagew - mleft - mright - 100 - 60}
	rows := [][]string{}

	// Init for title
	rows = append(rows, []string{"Timestamp", "UserID", "Activities"})
	for _, row := range activities {
		rows = append(rows, []string{row.Timestamp, row.UserID, row.Activities})
	}

	for _, row := range rows {
		curx, y := pdf.GetXY()
		x := curx

		height := 0.
		_, lineHt := pdf.GetFontSize()

		for i, txt := range row {
			lines := pdf.SplitLines([]byte(txt), cols[i])
			h := float64(len(lines))*lineHt + marginCell*float64(len(lines))
			if h > height {
				height = h
			}
		}

		// add a new page if the height of the row doesn't fit on the page
		if pdf.GetY()+height > pageh-mbottom {
			pdf.AddPage()
			y = pdf.GetY()
		}
		for i, txt := range row {
			width := cols[i]
			pdf.Rect(x, y, width, height, "")
			pdf.MultiCell(width, lineHt+marginCell, txt, "", "", false)
			x += width
			pdf.SetXY(x, y)
		}
		pdf.SetXY(curx, y+height)
	}

	t := time.Now().Format("20060102-150405")
	filepath = "./Activities-Report-" + t + ".pdf"

	fileStr := util.Filename(filepath)
	err := pdf.OutputFileAndClose(fileStr)
	util.Summary(err, fileStr)

	return filepath
}
