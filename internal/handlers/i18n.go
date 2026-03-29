package handlers

// i18nScript contains the full internationalization engine with Turkish and English translations.
// It provides:
// - t(key) function for translating strings in JS
// - DOM walker that auto-translates static text on page load
// - Language toggle (TR/EN) stored in localStorage
// - Default language: Turkish (tr)
const i18nScript = `
const I18N = (() => {
    const STORE_KEY = 'friday-lang';
    let lang = localStorage.getItem(STORE_KEY) || 'tr';

    const dict = {
        // ---- Common ----
        "Cancel": "İptal",
        "Save": "Kaydet",
        "Delete": "Sil",
        "Edit": "Düzenle",
        "Loading...": "Yükleniyor...",
        "Back": "Geri",
        "Error": "Hata",
        "View": "Görüntüle",

        // ---- Nav ----
        "Dashboard": "Panel",
        "Contacts": "Kişiler",
        "Groups": "Gruplar",
        "Drafts": "Taslaklar",
        "Send": "Gönder",
        "Batches": "Toplu Gönderim",
        "Checking...": "Kontrol ediliyor...",
        "Connected": "Bağlandı",
        "Disconnected": "Bağlantı kesildi",
        "Reconnecting...": "Yeniden bağlanıyor...",
        "Disconnecting...": "Bağlantı kesiliyor...",
        "Unknown": "Bilinmiyor",
        "Disconnect WhatsApp": "WhatsApp Bağlantısını Kes",

        // ---- Landing Page ----
        "WhatsApp API Server for Developers": "Geliştiriciler için WhatsApp API Sunucusu",
        "Ready to Connect": "Bağlanmaya Hazır",
        "Connect your WhatsApp account to start sending messages via API.": "API üzerinden mesaj göndermeye başlamak için WhatsApp hesabınızı bağlayın.",
        "Connect WhatsApp": "WhatsApp'ı Bağla",
        "Go to Dashboard": "Panele Git",
        "Connecting...": "Bağlanıyor...",
        "Checking session...": "Oturum kontrol ediliyor...",
        "Restoring session...": "Oturum geri yükleniyor...",
        "Session found, connecting...": "Oturum bulundu, bağlanıyor...",
        "Try Again": "Tekrar Dene",
        "QR Code Auth": "QR Kod Doğrulama",
        "Scan once, stay connected": "Bir kere tarayın, bağlı kalın",
        "REST API": "REST API",
        "Simple HTTP endpoints": "Basit HTTP uç noktaları",
        "Contact Sync": "Kişi Senkronizasyonu",
        "Access your contacts": "Kişilerinize erişin",
        "Secure": "Güvenli",
        "Local session storage": "Yerel oturum depolama",
        "Use responsibly. Not affiliated with WhatsApp Inc.": "Sorumlu kullanın. WhatsApp Inc. ile bağlantılı değildir.",
        "WhatsApp is ready. Access the dashboard to send messages.": "WhatsApp hazır. Mesaj göndermek için panele gidin.",

        // Landing page - JS dynamic
        "Connected! Going to dashboard...": "Bağlandı! Panele yönlendiriliyorsunuz...",
        "Session may be stale. Try disconnecting from your phone first.": "Oturum eskimiş olabilir. Önce telefonunuzdan bağlantıyı kesin.",
        "QR code required. Redirecting...": "QR kod gerekli. Yönlendiriliyor...",
        "Failed to connect: ": "Bağlanılamadı: ",

        // ---- QR Scan Page ----
        "Scan QR Code": "QR Kodu Tara",
        "QR Code Active": "QR Kod Aktif",
        "Scan with WhatsApp": "WhatsApp ile tarayın",
        "Refresh QR Code": "QR Kodu Yenile",
        "Expiring soon": "Süresi doluyor",
        "Expired": "Süresi doldu",
        "QR Code expired. Refreshing...": "QR Kodun süresi doldu. Yenileniyor...",
        "Connected! Redirecting to dashboard...": "Bağlandı! Panele yönlendiriliyorsunuz...",
        "WhatsApp connected successfully!": "WhatsApp başarıyla bağlandı!",
        "WhatsApp is already connected!": "WhatsApp zaten bağlı!",
        "Already connected! Redirecting to dashboard...": "Zaten bağlı! Panele yönlendiriliyorsunuz...",
        "Generating QR code...": "QR kod oluşturuluyor...",
        "Failed to connect. Please try again.": "Bağlanılamadı. Lütfen tekrar deneyin.",
        "Failed to refresh QR code": "QR kodu yenilenemedi",
        "How to scan": "Nasıl taranır",
        "Open WhatsApp on your phone": "Telefonunuzda WhatsApp'ı açın",
        "WhatsApp": "WhatsApp",
        "Settings": "Ayarlar",
        "Linked Devices": "Bağlı Cihazlar",
        "Link a Device": "Cihaz Bağla",
        "and scan this code": "ve bu kodu tarayın",

        // Nav - JS dynamic
        "Connection lost. Redirecting...": "Bağlantı kesildi. Yönlendiriliyor...",
        "Server connection lost. Redirecting...": "Sunucu bağlantısı kesildi. Yönlendiriliyor...",
        "Are you sure you want to disconnect WhatsApp? You will need to scan a new QR code to reconnect.": "WhatsApp bağlantısını kesmek istediğinize emin misiniz? Yeniden bağlanmak için yeni bir QR kod taramanız gerekecek.",
        "Disconnected successfully": "Bağlantı başarıyla kesildi",
        "Failed to disconnect": "Bağlantı kesilemedi",

        // ---- Dashboard ----
        "Send Message": "Mesaj Gönder",
        "Recipient": "Alıcı",
        "Phone number or contact name": "Telefon numarası veya kişi adı",
        "Message": "Mesaj",
        "Type your message...": "Mesajınızı yazın...",
        "Hello from Friday!": "Friday'den merhaba!",
        "Sending...": "Gönderiliyor...",
        "Response": "Yanıt",
        "API Reference": "API Referansı",
        "Get connection status": "Bağlantı durumunu al",
        "List all contacts": "Tüm kişileri listele",
        "Search contacts by name or phone": "Kişileri ada veya telefona göre ara",
        "Search contacts...": "Kişilerde ara...",
        "No contacts loaded": "Kişi yüklenmedi",
        "Load Contacts": "Kişileri Yükle",
        "No contacts found": "Kişi bulunamadı",
        "Connect WhatsApp to view contacts": "Kişileri görmek için WhatsApp'ı bağlayın",
        "Go to Connect": "Bağlantıya Git",
        "Error loading contacts": "Kişiler yüklenirken hata oluştu",
        "Please enter recipient and message": "Lütfen alıcı ve mesaj girin",
        "Message sent successfully!": "Mesaj başarıyla gönderildi!",
        "Failed to send: ": "Gönderilemedi: ",
        "Selected: ": "Seçildi: ",
        "View contact & attributes": "Kişi ve öznitelikleri görüntüle",
        "Failed to load contacts": "Kişiler yüklenemedi",
        "Send a message. Body: {\"recipient\": \"...\", \"message\": \"...\"}": "Mesaj gönder. Gövde: {\"recipient\": \"...\", \"message\": \"...\"}",

        // ---- Drafts Page ----
        "Manage your message templates": "Mesaj şablonlarınızı yönetin",
        "New Draft": "Yeni Taslak",
        "Using Placeholders": "Yer Tutucu Kullanımı",
        "No drafts yet": "Henüz taslak yok",
        "Create your first message template to get started": "Başlamak için ilk mesaj şablonunuzu oluşturun",
        "Create Draft": "Taslak Oluştur",
        "Edit Draft": "Taslak Düzenle",
        "Title": "Başlık",
        "e.g., Welcome Message": "örn., Hoş Geldiniz Mesajı",
        "Message Content": "Mesaj İçeriği",
        "Save Draft": "Taslağı Kaydet",
        "Use this draft": "Bu taslağı kullan",
        "Delete this draft?": "Bu taslak silinsin mi?",
        "Draft deleted": "Taslak silindi",
        "Draft updated": "Taslak güncellendi",
        "Draft created": "Taslak oluşturuldu",
        "Title and content are required": "Başlık ve içerik gereklidir",
        "Placeholders found: ": "Yer tutucular bulundu: ",
        "Failed to load drafts: ": "Taslaklar yüklenemedi: ",
        "Failed to load drafts": "Taslaklar yüklenemedi",
        "Failed to delete: ": "Silinemedi: ",
        "Failed to delete draft": "Taslak silinemedi",
        "Failed to save: ": "Kaydedilemedi: ",
        "Failed to save draft": "Taslak kaydedilemedi",

        // ---- Contact Detail Page ----
        "Send Message": "Mesaj Gönder",
        "Custom Attributes": "Özel Öznitelikler",
        "Add custom values to use in message placeholders": "Mesaj yer tutucularında kullanmak için özel değerler ekleyin",
        "Add Attribute": "Öznitelik Ekle",
        "Key (e.g., company)": "Anahtar (örn., şirket)",
        "Value": "Değer",
        "Loading attributes...": "Öznitelikler yükleniyor...",
        "No custom attributes yet": "Henüz özel öznitelik yok",
        "Add attributes to personalize messages with {{placeholders}}": "{{yer tutucular}} ile mesajları kişiselleştirmek için öznitelik ekleyin",
        "Contact not found": "Kişi bulunamadı",
        "Key and value are required": "Anahtar ve değer gereklidir",
        "Attribute saved": "Öznitelik kaydedildi",
        "Attribute updated": "Öznitelik güncellendi",
        "Attribute deleted": "Öznitelik silindi",
        "Failed to save attribute": "Öznitelik kaydedilemedi",
        "Failed to update attribute": "Öznitelik güncellenemedi",
        "Failed to delete attribute": "Öznitelik silinemedi",

        // ---- Send Page ----
        "Send a personalized message using a draft template": "Taslak şablon kullanarak kişiselleştirilmiş mesaj gönderin",
        "Select Draft": "Taslak Seç",
        "-- Choose a draft --": "-- Taslak seçin --",
        "Select Recipient": "Alıcı Seç",
        "Single Contact": "Tek Kişi",
        "Contact Group": "Kişi Grubu",
        "Edit contact attributes": "Kişi özniteliklerini düzenle",
        "Search contact groups...": "Kişi gruplarında ara...",
        "Edit group members": "Grup üyelerini düzenle",
        "Message Preview": "Mesaj Önizleme",
        "How the message will look with placeholders filled": "Mesaj yer tutucuları doldurulduğunda nasıl görünecek",
        "Select a draft and contact to preview": "Önizleme için taslak ve kişi seçin",
        "Select a draft and contact group to preview": "Önizleme için taslak ve kişi grubu seçin",
        "Batch Send Summary": "Toplu Gönderim Özeti",
        "Group:": "Grup:",
        "Recipients:": "Alıcılar:",
        "contacts": "kişi",
        "Draft:": "Taslak:",
        "Message Template": "Mesaj Şablonu",
        "Messages will be personalized using each contact's attributes": "Mesajlar her kişinin öznitelikleri kullanılarak kişiselleştirilecek",
        "Filled: ": "Doldurulmuş: ",
        "Missing: ": "Eksik: ",
        "Placeholders: ": "Yer tutucular: ",
        "Creating batch...": "Toplu gönderim oluşturuluyor...",
        "Failed to generate preview": "Önizleme oluşturulamadı",
        "Cannot send to an empty group": "Boş gruba gönderilemez",
        "Batch Created Successfully!": "Toplu Gönderim Başarıyla Oluşturuldu!",
        "Your messages have been queued and will be sent shortly.": "Mesajlarınız sıraya alındı ve kısa sürede gönderilecek.",
        "Stay Here": "Burada Kal",
        "View Progress": "İlerlemeyi Görüntüle",
        "No contacts available": "Kişi mevcut değil",
        "Failed to load contact groups": "Kişi grupları yüklenemedi",
        "Failed to send message": "Mesaj gönderilemedi",
        "Failed to create batch: ": "Toplu gönderim oluşturulamadı: ",
        "Failed to create batch": "Toplu gönderim oluşturulamadı",
        "Failed to load contacts": "Kişiler yüklenemedi",

        // ---- Contacts Page ----
        "Manage contacts and their custom attributes for message personalization": "Mesaj kişiselleştirme için kişileri ve özel özniteliklerini yönetin",
        "Search contacts by name or phone...": "Ada veya telefona göre kişi arayın...",
        "Attribute keys in use:": "Kullanılan öznitelik anahtarları:",
        "Loading contacts...": "Kişiler yükleniyor...",
        "Connect WhatsApp to see your contacts": "Kişilerinizi görmek için WhatsApp'ı bağlayın",
        "View attributes": "Öznitelikleri görüntüle",

        // ---- Groups Page ----
        "Contact Groups": "Kişi Grupları",
        "Organize contacts into groups for batch messaging": "Toplu mesajlaşma için kişileri gruplara düzenleyin",
        "New Group": "Yeni Grup",
        "No groups yet": "Henüz grup yok",
        "Create your first contact group to start batch messaging": "Toplu mesajlaşmaya başlamak için ilk kişi grubunuzu oluşturun",
        "Create Group": "Grup Oluştur",
        "Edit Group": "Grubu Düzenle",
        "Loading groups...": "Gruplar yükleniyor...",
        "Group Name": "Grup Adı",
        "e.g., VIP Customers": "örn., VIP Müşteriler",
        "Manage Members": "Üyeleri Yönet",
        "members": "üye",
        "Group deleted": "Grup silindi",
        "Group updated": "Grup güncellendi",
        "Group created": "Grup oluşturuldu",
        "Group name is required": "Grup adı gereklidir",
        "Failed to load groups": "Gruplar yüklenemedi",
        "Failed to delete group": "Grup silinemedi",
        "Failed to save group": "Grup kaydedilemedi",

        // ---- Group Detail Page ----
        "Back to Groups": "Gruplara Dön",
        "Send to Group": "Gruba Gönder",
        "Add Members": "Üye Ekle",
        "Add Selected": "Seçilenleri Ekle",
        "Members": "Üyeler",
        "No members in this group yet": "Bu grupta henüz üye yok",
        "Select a draft...": "Taslak seçin...",
        "Messages will be sent with 10-15 second random delays to avoid spam detection.": "Spam algılanmasını önlemek için mesajlar 10-15 saniye rastgele aralıklarla gönderilecek.",
        "Start Batch": "Toplu Gönderimi Başlat",
        "Members added": "Üyeler eklendi",
        "Remove this member?": "Bu üye kaldırılsın mı?",
        "Member removed": "Üye kaldırıldı",
        "Add members first": "Önce üye ekleyin",
        "Failed to add members": "Üyeler eklenemedi",
        "Failed to remove member": "Üye kaldırılamadı",
        "Failed to start batch": "Toplu gönderim başlatılamadı",
        "Failed to load group": "Grup yüklenemedi",

        // ---- Batch Runs Page ----
        "Batch Runs": "Toplu Gönderimler",
        "Track and manage batch message sending": "Toplu mesaj gönderimini takip edin ve yönetin",
        "Batch in Progress": "Toplu Gönderim Devam Ediyor",
        "Batch": "Gönderim",
        "Status": "Durum",
        "Progress": "İlerleme",
        "Created": "Oluşturulma",
        "Actions": "İşlemler",
        "No batch runs yet": "Henüz toplu gönderim yok",
        "Start by creating a group and sending a draft to it": "Bir grup oluşturarak ve taslak göndererek başlayın",
        "Go to Groups": "Gruplara Git",
        "Cancel this batch?": "Bu toplu gönderim iptal edilsin mi?",
        "Batch cancelled": "Toplu gönderim iptal edildi",
        "Delete this batch run?": "Bu toplu gönderim silinsin mi?",
        "Batch deleted": "Toplu gönderim silindi",
        "Failed to load batches": "Toplu gönderimler yüklenemedi",
        "Failed to cancel batch": "Toplu gönderim iptal edilemedi",
        "Failed to delete batch": "Toplu gönderim silinemedi",
        "to": "→",

        // Batch status labels
        "Queued": "Sırada",
        "Running": "Çalışıyor",
        "Completed": "Tamamlandı",
        "Cancelled": "İptal Edildi",
        "Failed": "Başarısız",

        // ---- Batch Detail Page ----
        "Back to Batches": "Toplu Gönderimlere Dön",
        "Loading": "Yükleniyor",
        "sent": "gönderildi",
        "failed": "başarısız",
        "Sending to": "Gönderiliyor:",
        "Next message in": "Sonraki mesaj",
        "seconds": "saniye içinde",
        "Cancel Batch": "Toplu Gönderimi İptal Et",
        "Message History": "Mesaj Geçmişi",
        "No messages sent yet": "Henüz mesaj gönderilmedi",
        "Message Details": "Mesaj Detayları",
        "Sent Message": "Gönderilen Mesaj",
        "Batch completed!": "Toplu gönderim tamamlandı!",
        "Batch not found": "Toplu gönderim bulunamadı",

        // Placeholder text
        "Use {{name}} syntax in your drafts.": "Taslaklarınızda {{name}} söz dizimini kullanın.",
        "Built-in:": "Yerleşik:",
        "Custom: Any attribute you add to a contact.": "Özel: Bir kişiye eklediğiniz herhangi bir öznitelik.",

        // Misc dynamic
        "No contacts found matching": "Eşleşen kişi bulunamadı:",
        "No groups found matching": "Eşleşen grup bulunamadı:",
        "No contact groups yet.": "Henüz kişi grubu yok.",
        "Create one": "Oluşturun",
        "Tap": "Dokunun",
        "Go to": "Git:",
        "Open": "Açın",
        "on your phone": "telefonunuzda"
    };

    function t(text) {
        if (lang === 'en' || !text) return text;
        // Direct match
        if (dict[text]) return dict[text];
        return text;
    }

    function setLang(newLang) {
        lang = newLang;
        localStorage.setItem(STORE_KEY, lang);
        location.reload();
    }

    function getLang() { return lang; }

    function translateDOM() {
        if (lang === 'en') return;

        document.documentElement.lang = lang;

        // Walk text nodes
        const walker = document.createTreeWalker(
            document.body,
            NodeFilter.SHOW_TEXT,
            {
                acceptNode(node) {
                    const p = node.parentElement;
                    if (!p) return NodeFilter.FILTER_REJECT;
                    const tag = p.tagName;
                    if (['SCRIPT','STYLE','CODE','PRE','TEXTAREA'].includes(tag)) return NodeFilter.FILTER_REJECT;
                    if (p.closest('code, pre, textarea, script, style')) return NodeFilter.FILTER_REJECT;
                    return NodeFilter.FILTER_ACCEPT;
                }
            }
        );

        const nodes = [];
        while (walker.nextNode()) nodes.push(walker.currentNode);

        nodes.forEach(node => {
            const text = node.textContent;
            const trimmed = text.trim();
            if (!trimmed) return;
            if (dict[trimmed]) {
                node.textContent = text.replace(trimmed, dict[trimmed]);
            }
        });

        // Translate placeholders
        document.querySelectorAll('input[placeholder], textarea[placeholder]').forEach(el => {
            if (dict[el.placeholder]) el.placeholder = dict[el.placeholder];
        });

        // Translate title attributes
        document.querySelectorAll('[title]').forEach(el => {
            if (dict[el.title]) el.title = dict[el.title];
        });
    }

    // Run on DOM ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', translateDOM);
    } else {
        translateDOM();
    }

    return { t, setLang, getLang, translateDOM };
})();
const t = I18N.t;
`

// langSwitcher is the HTML for the language toggle button, placed in the nav bar
const langSwitcher = `
<div class="flex items-center border border-gray-200 rounded-lg overflow-hidden text-xs font-medium">
    <button onclick="I18N.setLang('tr')" id="lang-tr"
        class="px-2 py-1 transition-colors">TR</button>
    <button onclick="I18N.setLang('en')" id="lang-en"
        class="px-2 py-1 transition-colors">EN</button>
</div>
<script>
(function(){
    var l = I18N.getLang();
    var trBtn = document.getElementById('lang-tr');
    var enBtn = document.getElementById('lang-en');
    if (l === 'tr') {
        trBtn.className = 'px-2 py-1 transition-colors bg-whatsapp-500 text-white';
        enBtn.className = 'px-2 py-1 transition-colors text-gray-500 hover:bg-gray-100';
    } else {
        enBtn.className = 'px-2 py-1 transition-colors bg-whatsapp-500 text-white';
        trBtn.className = 'px-2 py-1 transition-colors text-gray-500 hover:bg-gray-100';
    }
})();
</script>
`

// landingLangSwitcher is the language toggle for pages without the nav bar (landing, QR scan)
const landingLangSwitcher = `
<div class="fixed top-4 right-4 z-50 flex items-center border border-white/30 rounded-lg overflow-hidden text-xs font-medium bg-white/20 backdrop-blur-sm">
    <button onclick="I18N.setLang('tr')" id="lang-tr"
        class="px-2.5 py-1.5 transition-colors">TR</button>
    <button onclick="I18N.setLang('en')" id="lang-en"
        class="px-2.5 py-1.5 transition-colors">EN</button>
</div>
<script>
(function(){
    var l = I18N.getLang();
    var trBtn = document.getElementById('lang-tr');
    var enBtn = document.getElementById('lang-en');
    if (l === 'tr') {
        trBtn.className = 'px-2.5 py-1.5 transition-colors bg-whatsapp-500 text-white';
        enBtn.className = 'px-2.5 py-1.5 transition-colors text-white/70 hover:text-white';
    } else {
        enBtn.className = 'px-2.5 py-1.5 transition-colors bg-whatsapp-500 text-white';
        trBtn.className = 'px-2.5 py-1.5 transition-colors text-white/70 hover:text-white';
    }
})();
</script>
`
