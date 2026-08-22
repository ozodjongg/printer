# 🖨️ Telegram Auto Printer (Go)

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat&logo=go)](https://golang.org)
[![Platform](https://img.shields.io/badge/Platform-Windows-0078D6?style=flat&logo=windows)](https://microsoft.com)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

Telefoningizdan Telegram bot orqali yuborilgan har qanday hujjat (PDF, Word, TXT) va rasmlarni kompyuterga avtomatik yuklab olib, ulangan printerda darhol chop etuvchi yengil, tezkor va xavfsiz Go utilitasi.

---

## ✨ Xususiyatlar

- 🚀 **Nol konfiguratsiya (Zero-config setup):** Birinchi marta ishga tushganda `telegram.json` faylini interaktiv tarzda o'zi yaratadi.
- 🔒 **Parol orqali avtorizatsiya:** Noma'lum foydalanuvchilar chop etishidan himoyalangan. Parolni kiritgan foydalanuvchilar avtomatik ruxsat oladi.
- 💾 **Dinamik saqlash:** Ruxsat berilgan foydalanuvchilar ID'si avtomatik ravishda `telegram.json` fayliga saqlanadi.
- 🖨️ **Windows Native Print:** Tashqi og'ir dasturlarsiz, Windows'ning ichki `ShellExecuteW` WinAPI mexanizmi orqali fonda chop etadi.
- ⚡ **Yuqori tezlik va yengillik:** Go tilida yozilgani sababli tizim resurslarini minimal darajada sarflaydi.

---

## 📋 Talablar

- **Operatsion tizim:** Windows (7 / 8.1 / 10 / 11)
- **Printer:** Kompyuterga ulangan va sukut bo'yicha (*default*) qilib belgilangan printer.
- **Go SDK:** (Faqat manba kodidan yig'uvchilar uchun Go 1.20+)

---

## 🚀 O'rnatish va Ishga Tushirish

### 1. Tayyor Binary (.exe) orqali
1. [Releases](../../releases) bo'limidan eng so'nggi `telegram-printer-win-amd64.exe` faylini yuklab oling.
2. `.exe` faylini alohida papkaga joylang va uni ishga tushiring.
3. Konsol oynasida so'ralgan **Bot Token** va **Parol**ni kiriting.
4. Dastur avtomatik ravishda `telegram.json` faylini yaratadi va botni ishga tushiradi.

### 2. Manba kodidan (Source Code)
```bash
# Repozitoriyani klonlash
git clone [https://github.com/ozodjongg/printer.git](https://github.com/ozodjongg/printer.git)
cd printer

# Bog'liqliklarni yuklab olish
go mod download

# Ishga tushirish
go run main.go