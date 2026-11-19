<div align="center">
  <h1 style="border:none; margin-top: 10px; margin-bottom: 6px;">Nouriko</h1>
  <p><strong>Chatbot AI yang membantu anak kos memilih makanan bergizi dari bahan dan uang yang mereka punya</strong></p>

  <p>
    <a href="https://github.com/Xuerns/Nouriko/actions/workflows/ci.yml"><img src="https://github.com/Xuerns/Nouriko/actions/workflows/ci.yml/badge.svg" alt="CI Status"></a>
    <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.27+-00ADD8?logo=go&logoColor=white" alt="Go Version"></a>
    <a href="https://nextjs.org/"><img src="https://img.shields.io/badge/Next.js-16-black?logo=nextdotjs&logoColor=white" alt="Next.js 16"></a>
    <a href="https://www.postgresql.org/"><img src="https://img.shields.io/badge/PostgreSQL-18-4169E1?logo=postgresql&logoColor=white" alt="PostgreSQL 18"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/License-AGPL--3.0-blue.svg" alt="License: AGPL-3.0"></a>
  </p>
</div>

## Overview

Anak kos biasanya mengatur makan sendiri dengan uang pas-pasan dan dapur seadanya, sering sambil mengejar jadwal yang padat. Banyak yang akhirnya bergantung pada makanan instan atau melewatkan jam makan, dan jarang ada yang tahu berapa gizi yang sebenarnya mereka dapat dari makanan sehari-hari.

Di Nouriko, pengguna cukup menulis bahan yang ada di kamar, menyebut nominal uang, atau mengirim foto makanan lewat satu kolom chat. AI membalas dengan rekomendasi makanan beserta skor kesehatan, rincian makro, dan perkiraan harganya. Makanan yang dipilih bisa disimpan, jadi pengguna bisa melihat total asupan hari itu dibanding target kalori pribadinya.

## Quick Start

1. Clone repository ini.

   ```bash
   git clone https://github.com/Xuerns/Nouriko.git
   cd Nouriko
   ```

2. Salin file environment, lalu ganti `POSTGRES_PASSWORD` dengan password yang kuat.

   ```bash
   cp .env.example .env
   ```

3. Build dan jalankan semua service.

   ```bash
   docker compose up -d --build
   ```

## License

This project is licensed under the GNU Affero General Public License v3.0 (AGPL-3.0). See the [LICENSE](LICENSE) file for details.