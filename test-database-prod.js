// ============================================
// SCRIPT POUR TESTER LA BASE DE DONNÉES EN PRODUCTION
// ============================================
// Copiez-collez TOUT ce code dans la console de l'inspecteur (F12)
// ============================================

(async function() {
    console.log('%c🔍 Vérification de la base de données...', 'font-size: 16px; font-weight: bold; color: #4CAF50;');
    console.log('');
    
    try {
        // Récupérer l'URL de base depuis la page actuelle
        const baseURL = window.location.origin;
        const apiURL = `${baseURL}/api/database/info`;
        
        console.log(`📡 Connexion à: ${apiURL}`);
        console.log('');
        
        const response = await fetch(apiURL);
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        const data = await response.json();
        
        console.log('%c📊 Informations de la base de données:', 'font-size: 14px; font-weight: bold;');
        console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
        
        if (data.type === 'MySQL') {
            console.log('%c✅ Type: MySQL', 'color: #4CAF50; font-weight: bold;');
            console.log(`   Host: ${data.host}`);
            console.log(`   Port: ${data.port}`);
            console.log(`   Database: ${data.database}`);
            console.log(`   User: ${data.user}`);
            
            if (data.stats) {
                console.log('');
                console.log('%c📈 Statistiques:', 'font-weight: bold;');
                console.log(`   Agents: ${data.stats.agents || 0}`);
                console.log(`   Commandes: ${data.stats.commands || 0}`);
                console.log(`   Fichiers: ${data.stats.files || 0}`);
                console.log(`   Imprimantes: ${data.stats.printers || 0}`);
            }
            
            console.log('');
            console.log('%c✅ CONFIRMATION: Le serveur utilise bien MySQL !', 'color: #4CAF50; font-weight: bold; font-size: 14px;');
        } else if (data.type === 'SQLite') {
            console.log('%c⚠️  Type: SQLite', 'color: #FF9800; font-weight: bold;');
            console.log(`   Chemin: ${data.path}`);
            
            if (data.stats) {
                console.log('');
                console.log('%c📈 Statistiques:', 'font-weight: bold;');
                console.log(`   Agents: ${data.stats.agents || 0}`);
                console.log(`   Commandes: ${data.stats.commands || 0}`);
                console.log(`   Fichiers: ${data.stats.files || 0}`);
                console.log(`   Imprimantes: ${data.stats.printers || 0}`);
            }
            
            console.log('');
            console.log('%c⚠️  ATTENTION: Le serveur utilise SQLite (pas MySQL) !', 'color: #FF9800; font-weight: bold; font-size: 14px;');
        } else {
            console.log('%c❌ Type inconnu:', 'color: #F44336; font-weight: bold;', data.type);
            console.log('Données complètes:', data);
        }
        
        console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
        
        // Retourner les données pour inspection
        return data;
        
    } catch (error) {
        console.error('%c❌ Erreur lors de la vérification:', 'color: #F44336; font-weight: bold;', error);
        console.log('');
        console.log('💡 Vérifications alternatives:');
        console.log('1. Vérifiez que vous êtes bien connecté à la production');
        console.log('2. Essayez cette commande simple:');
        console.log('   fetch("/api/database/info").then(r => r.json()).then(console.log)');
        console.log('');
        throw error;
    }
})();

