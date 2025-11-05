// Script JavaScript à copier-coller dans la console de l'inspecteur du navigateur
// Pour vérifier quelle base de données est utilisée

(async function() {
    console.log('🔍 Vérification de la base de données...\n');
    
    try {
        const response = await fetch('/api/database/info');
        const data = await response.json();
        
        console.log('📊 Informations de la base de données:');
        console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
        
        if (data.type === 'MySQL') {
            console.log('✅ Type: MySQL');
            console.log(`   Host: ${data.host}`);
            console.log(`   Port: ${data.port}`);
            console.log(`   Database: ${data.database}`);
            console.log(`   User: ${data.user}`);
            
            if (data.stats) {
                console.log('\n📈 Statistiques:');
                console.log(`   Agents: ${data.stats.agents || 0}`);
                console.log(`   Commandes: ${data.stats.commands || 0}`);
                console.log(`   Fichiers: ${data.stats.files || 0}`);
                console.log(`   Imprimantes: ${data.stats.printers || 0}`);
            }
            
            console.log('\n✅ Le serveur utilise bien MySQL !');
        } else if (data.type === 'SQLite') {
            console.log('⚠️  Type: SQLite');
            console.log(`   Chemin: ${data.path}`);
            
            if (data.stats) {
                console.log('\n📈 Statistiques:');
                console.log(`   Agents: ${data.stats.agents || 0}`);
                console.log(`   Commandes: ${data.stats.commands || 0}`);
                console.log(`   Fichiers: ${data.stats.files || 0}`);
                console.log(`   Imprimantes: ${data.stats.printers || 0}`);
            }
            
            console.log('\n⚠️  Le serveur utilise SQLite (pas MySQL) !');
        } else {
            console.log('❌ Type inconnu:', data.type);
        }
        
        console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
        
    } catch (error) {
        console.error('❌ Erreur lors de la vérification:', error);
        console.log('\n💡 Essayez cette commande alternative:');
        console.log('fetch("/api/database/info").then(r => r.json()).then(console.log)');
    }
})();

