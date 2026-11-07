allprojects {
    repositories {
        google()
        mavenCentral()
    }
}

val newBuildDir: Directory =
    rootProject.layout.buildDirectory
        .dir("../../build")
        .get()
rootProject.layout.buildDirectory.value(newBuildDir)

subprojects {
    val newSubprojectBuildDir: Directory = newBuildDir.dir(project.name)
    project.layout.buildDirectory.value(newSubprojectBuildDir)
}
// Windows: proje (ör. D:) ile pub önbelleği (C:\Users\…\Pub\Cache) farklı
// sürücülerdeyse Kotlin'in artımlı derlemesi eklenti kaynaklarına göreli yol
// kuramaz ve "Could not close incremental caches" ile düşer. Proje klasörü
// dışındaki (pub önbelleğindeki eklenti) modüllerde artımlı derleme kapatılır;
// :app etkilenmez. Eklenti kodu küçük olduğundan derleme süresine etkisi azdır.
subprojects {
    if (!projectDir.canonicalPath.startsWith(rootDir.canonicalPath)) {
        extra["kotlin.incremental"] = "false"
    }
}
subprojects {
    project.evaluationDependsOn(":app")
}

tasks.register<Delete>("clean") {
    delete(rootProject.layout.buildDirectory)
}
