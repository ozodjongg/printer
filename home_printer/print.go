package home_printer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

type PrintOptions struct {
	FilePath  string
	Mode      string
	Color     string
	Duplex    string
	Copies    int
	PageRange string // Masalan: "5-11" yoki "1,3,5"
}

var (
	printMutex        sync.Mutex
	shell32           = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteW = shell32.NewProc("ShellExecuteW")
)

func getAppDir() string {
	execPath, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(execPath)
}

func Print(opt PrintOptions) error {
	printMutex.Lock()
	defer printMutex.Unlock()

	absPath, err := filepath.Abs(opt.FilePath)
	if err != nil {
		return fmt.Errorf("fayl yo'lini aniqlashda xatolik: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(absPath))
	workingPath := absPath

	var filesToDelete []string
	filesToDelete = append(filesToDelete, absPath)

	defer func() {
		go func(files []string) {
			time.Sleep(5 * time.Second)
			for _, f := range files {
				os.Remove(f)
			}
		}(filesToDelete)
	}()

	appDir := getAppDir()
	sumatraExe := filepath.Join(appDir, "SumatraPDF.exe")
	pdfcpuExe := filepath.Join(appDir, "pdfcpu.exe")

	if _, err := os.Stat(sumatraExe); os.IsNotExist(err) {
		sumatraExe = "SumatraPDF.exe"
	}
	if _, err := os.Stat(pdfcpuExe); os.IsNotExist(err) {
		pdfcpuExe = "pdfcpu.exe"
	}

	// 1. Rasmlarni PDF formatga o'tkazish
	if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".bmp" {
		pdfPath := absPath + ".pdf"
		cmd := exec.Command(pdfcpuExe, "import", pdfPath, absPath)
		if err := cmd.Run(); err == nil {
			workingPath = pdfPath
			filesToDelete = append(filesToDelete, pdfPath)
		}
	}

	// 2. Sahifalarni saralab qirqib olish (Trim Page Range)
	if opt.PageRange != "" {
		trimmedPath := workingPath + "_trimmed.pdf"
		cmd := exec.Command(pdfcpuExe, "trim", "-pages", opt.PageRange, workingPath, trimmedPath)
		if err := cmd.Run(); err == nil {
			workingPath = trimmedPath
			filesToDelete = append(filesToDelete, trimmedPath)
		}
	}

	// 3. 2-in-1 yoki Booklet rejimlari (Trimlangan sahifalarga nisbatan qo'llaniladi)
	if opt.Mode == "2up" {
		outPath := workingPath + "_2up.pdf"
		cmd := exec.Command(pdfcpuExe, "nup", "--", "form:A4", outPath, "2", workingPath)
		if err := cmd.Run(); err == nil {
			workingPath = outPath
			filesToDelete = append(filesToDelete, outPath)
		}
	} else if opt.Mode == "booklet" {
		outPath := workingPath + "_booklet.pdf"
		cmd := exec.Command(pdfcpuExe, "booklet", "--", "form:A4", outPath, "2", workingPath)
		if err := cmd.Run(); err == nil {
			workingPath = outPath
			filesToDelete = append(filesToDelete, outPath)
		}
	}

	// 4. SumatraPDF orqali chop etish
	if _, err := os.Stat(sumatraExe); err == nil || sumatraExe == "SumatraPDF.exe" {
		settings := fmt.Sprintf("%s,%s,%dx", opt.Color, opt.Duplex, opt.Copies)
		if opt.PageRange != "" && opt.Mode == "1up" {
			settings = fmt.Sprintf("%s,%s", opt.PageRange, settings)
		}
		cmd := exec.Command(sumatraExe, "-print-to-default", "-silent", "-print-settings", settings, workingPath)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	// 5. Fallback: Windows Shell
	verb, _ := syscall.UTF16PtrFromString("print")
	file, _ := syscall.UTF16PtrFromString(workingPath)

	ret, _, _ := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		0, 0, 0,
	)

	if ret <= 32 {
		return fmt.Errorf("windows shell chop etishda xatolik (kod: %d)", ret)
	}
	return nil
}