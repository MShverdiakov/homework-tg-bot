# Telegram Homework Bot

Этот бот создан для проверки выполнения домашнего задания моей сестры. Он загружает расписание и домашнюю работу в базу данных MongoDB, а затем отправляет уведомления родителям о статусе выполнения.

## 🚀 Функционал
- **Добавление ученика** через команду `/addstudent`, указав родителя, который получит уведомления.
- **Хранение расписания** и списка домашних заданий в MongoDB.
- **Автоматическая проверка домашнего задания в 22:00** и отправка уведомления родителю, если домашка не сделана.
- **Отправка фото домашнего задания**, если оно было загружено учеником.
- **Возможность ручной проверки** выполнения через команду `/checkhw`.
- **Родитель получает уведомления** о статусе выполнения домашнего задания.

## 📦 Хранение данных в MongoDB
Бот сохраняет следующую информацию в базе данных:
- **Ученики** (ID, имя пользователя, родительский контакт, расписание).
- **Расписание** (дни недели, предметы, список домашних заданий).
- **Домашние задания** (название предмета, фото, статус выполнения).

## 🛠️ Технологии
- **Язык:** Go
- **База данных:** MongoDB
- **Бот:** Telegram Bot API
- **Деплой:** Docker + Raspberry Pi OS

## 🔧 Установка и запуск
### 1. Клонирование репозитория
```bash
git clone https://github.com/MShverdiakov/homework-tg-bot.git
cd homework-tg-bot
```

### 2. Запуск через Docker
```bash
docker build -t homework-bot .
docker run -d --name homework-bot --restart always --env-file .env homework-bot
```

## 📌 Переменные окружения
Создайте `.env` файл и укажите:
```env
TELEGRAM_BOT_TOKEN=your_bot_token
MONGO_URI=mongodb://your_mongo_db
```

## 📞 Контакты
Автор: [MShverdiakov](https://github.com/MShverdiakov)

