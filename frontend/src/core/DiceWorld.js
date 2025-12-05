import * as THREE from 'three';
import * as CANNON from 'cannon-es';
import { EffectComposer } from 'three/addons/postprocessing/EffectComposer.js';
import { RenderPixelatedPass } from 'three/addons/postprocessing/RenderPixelatedPass.js';
import { OutputPass } from 'three/addons/postprocessing/OutputPass.js';
import { UnrealBloomPass } from 'three/addons/postprocessing/UnrealBloomPass.js';

export class DiceWorld {
    constructor(canvas) {
        this.canvas = canvas;
        this.width = window.innerWidth;
        this.height = window.innerHeight;
        this.theme = 'clean';

        // 1. Setup Three.js
        this.scene = new THREE.Scene();
        
        // Камера чуть ниже и ближе, чтобы создать ощущение присутствия за столом
        this.camera = new THREE.PerspectiveCamera(35, this.width / this.height, 0.1, 100);
        this.camera.position.set(0, 20, 15);
        this.camera.lookAt(0, 0, 0);

        this.renderer = new THREE.WebGLRenderer({ 
            canvas: this.canvas, 
            antialias: false, // Выключаем сглаживание для ретро-эффекта
            alpha: true 
        });
        this.renderer.setSize(this.width, this.height);
        this.renderer.shadowMap.enabled = true;
        this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;

        // 2. Post-Processing (Магия Клубники)
        this.composer = new EffectComposer(this.renderer);
        
        // Эффект Пикселизации (Рендер в разрешении / 4)
        // pixelSize = 3 (довольно крупные пиксели)
        this.pixelPass = new RenderPixelatedPass(3, this.scene, this.camera);
        this.pixelPass.normalEdgeStrength = 0; // Убираем обводку краев
        this.pixelPass.depthEdgeStrength = 0;
        this.composer.addPass(this.pixelPass);

        // Блум (свечение) для атмосферы
        this.bloomPass = new UnrealBloomPass(new THREE.Vector2(this.width, this.height), 0.5, 0.4, 0.85);
        this.bloomPass.enabled = false; // Включаем только в индастриал
        this.composer.addPass(this.bloomPass);

        this.outputPass = new OutputPass();
        this.composer.addPass(this.outputPass);

        // 3. Physics
        this.world = new CANNON.World();
        this.world.gravity.set(0, -30, 0); // Тяжелая гравитация
        this.world.broadphase = new CANNON.NaiveBroadphase();
        this.world.solver.iterations = 10;

        // Materials Physics
        this.diceMaterialPhysics = new CANNON.Material();
        this.floorMaterialPhysics = new CANNON.Material();
        const contactMaterial = new CANNON.ContactMaterial(
            this.floorMaterialPhysics, this.diceMaterialPhysics,
            { friction: 0.05, restitution: 0.4 }
        );
        this.world.addContactMaterial(contactMaterial);

        // 4. Objects
        // Генерируем "шумные" текстуры
        this.noiseTexture = this.createNoiseTexture();
        this.createFloor();
        this.createLights();

        this.diceMeshes = [];
        this.diceBodies = [];

        window.addEventListener('resize', () => this.onResize());
    }

    // Хелпер для генерации грязной текстуры (чтобы не качать картинки)
    createNoiseTexture() {
        const size = 512;
        const canvas = document.createElement('canvas');
        canvas.width = size;
        canvas.height = size;
        const ctx = canvas.getContext('2d');
        
        ctx.fillStyle = '#444';
        ctx.fillRect(0,0, size, size);

        for (let i = 0; i < 40000; i++) {
            const x = Math.random() * size;
            const y = Math.random() * size;
            const gray = Math.floor(Math.random() * 50);
            ctx.fillStyle = `rgba(${gray},${gray},${gray},0.1)`;
            ctx.fillRect(x, y, 2, 2);
        }

        const texture = new THREE.CanvasTexture(canvas);
        texture.wrapS = THREE.RepeatWrapping;
        texture.wrapT = THREE.RepeatWrapping;
        return texture;
    }

    createFloor() {
        // Пол - это стол
        const geometry = new THREE.PlaneGeometry(100, 100);
        
        // Материал стола по умолчанию
        this.floorMaterial = new THREE.MeshStandardMaterial({ 
            color: 0xffffff, 
            roughness: 0.8,
            metalness: 0.1 
        });

        this.floorMesh = new THREE.Mesh(geometry, this.floorMaterial);
        this.floorMesh.rotation.x = -Math.PI / 2;
        this.floorMesh.receiveShadow = true;
        this.scene.add(this.floorMesh);

        // Physics Floor
        const floorShape = new CANNON.Plane();
        const floorBody = new CANNON.Body({ mass: 0, material: this.floorMaterialPhysics });
        floorBody.addShape(floorShape);
        floorBody.quaternion.setFromAxisAngle(new CANNON.Vec3(1, 0, 0), -Math.PI / 2);
        this.world.addBody(floorBody);
    }

    createLights() {
        // 1. Общий мягкий свет (Clean theme)
        this.ambientLight = new THREE.AmbientLight(0xffffff, 0.6);
        this.scene.add(this.ambientLight);

        // 2. Солнце (Clean theme)
        this.dirLight = new THREE.DirectionalLight(0xffffff, 0.8);
        this.dirLight.position.set(10, 20, 10);
        this.dirLight.castShadow = true;
        this.dirLight.shadow.mapSize.width = 2048;
        this.dirLight.shadow.mapSize.height = 2048;
        this.scene.add(this.dirLight);
        
        // 3. Лампа над столом (Industrial Theme)
        // Теплый, жесткий свет, висящий прямо над центром
        this.industrialLight = new THREE.PointLight(0xffaa55, 0, 100); // Изначально выключен (intensity 0)
        this.industrialLight.position.set(0, 15, 0);
        this.industrialLight.castShadow = true;
        this.industrialLight.shadow.bias = -0.0001;
        this.scene.add(this.industrialLight);

        // Подсветка снизу (для драматизма)
        this.rimLight = new THREE.PointLight(0x445566, 0, 50);
        this.rimLight.position.set(-10, 5, -10);
        this.scene.add(this.rimLight);
    }

    setTheme(themeName) {
        this.theme = themeName;
        
        if (themeName === 'industrial') {
            // == INDUSTRIAL MODE ==
            
            // 1. Фон не черный, а темно-серый "бетон"
            this.scene.background = new THREE.Color(0x1a1816);

            // 2. Свет: Выключаем солнце, включаем Лампу
            this.ambientLight.intensity = 0.1;
            this.dirLight.intensity = 0;
            
            // Лампа светит ярко, создавая жесткие тени
            this.industrialLight.intensity = 80;
            this.rimLight.intensity = 10;

            // 3. Материал стола: Грязный металл
            this.floorMaterial.map = this.noiseTexture;
            this.floorMaterial.color.setHex(0x555555);
            this.floorMaterial.roughness = 0.4;
            this.floorMaterial.metalness = 0.6;
            this.floorMaterial.needsUpdate = true;

            // 4. Post-Processing: Включаем ретро-эффекты
            this.pixelPass.setPixelSize(3); // Крупный пиксель
            this.bloomPass.enabled = true;  // Легкое свечение

        } else {
            // == CLEAN MODE ==
            this.scene.background = null; // Прозрачный

            this.ambientLight.intensity = 0.6;
            this.dirLight.intensity = 0.8;
            this.industrialLight.intensity = 0;
            this.rimLight.intensity = 0;

            this.floorMaterial.map = null;
            this.floorMaterial.color.setHex(0xffffff);
            this.floorMaterial.roughness = 0.8;
            this.floorMaterial.metalness = 0.0;
            this.floorMaterial.needsUpdate = true;

            this.pixelPass.setPixelSize(1); // Обычное разрешение
            this.bloomPass.enabled = false;
        }

        // Обновляем существующие кубики под тему
        this.updateDiceMaterials();
    }

    updateDiceMaterials() {
        this.diceMeshes.forEach(mesh => {
            if (this.theme === 'industrial') {
                mesh.material.color.setHex(0xcc3300); // Ржавый красный/оранжевый
                mesh.material.emissive.setHex(0x220500); // Слабое свечение изнутри
                mesh.material.roughness = 0.3;
                mesh.material.metalness = 0.8;
                mesh.material.map = this.noiseTexture; // Добавляем шум на сам кубик
            } else {
                mesh.material.color.setHex(0x34d399); // Clean Green
                mesh.material.emissive.setHex(0x000000);
                mesh.material.roughness = 0.2;
                mesh.material.metalness = 0.1;
                mesh.material.map = null;
            }
            mesh.material.needsUpdate = true;
        });
    }

    addDice() {
        const size = 1.5;
        const geometry = new THREE.BoxGeometry(size, size, size);
        const material = new THREE.MeshStandardMaterial();
        
        const mesh = new THREE.Mesh(geometry, material);
        mesh.castShadow = true;
        mesh.receiveShadow = true;
        this.scene.add(mesh);
        this.diceMeshes.push(mesh);

        // Physics
        const shape = new CANNON.Box(new CANNON.Vec3(size/2, size/2, size/2));
        const body = new CANNON.Body({ mass: 5, material: this.diceMaterialPhysics }); // Увеличили массу
        body.addShape(shape);
        body.position.set((Math.random() - 0.5) * 5, 12, (Math.random() - 0.5) * 5);
        body.angularVelocity.set(Math.random() * 10, Math.random() * 10, Math.random() * 10);
        
        this.world.addBody(body);
        this.diceBodies.push(body);

        // Сразу красим в нужный цвет
        this.updateDiceMaterials();
    }

    clearDice() {
        for(let mesh of this.diceMeshes) {
            mesh.geometry.dispose();
            mesh.material.dispose();
            this.scene.remove(mesh);
        }
        for(let body of this.diceBodies) this.world.removeBody(body);
        this.diceMeshes = [];
        this.diceBodies = [];
    }

    throwDice(count = 1) {
        this.clearDice();
        for (let i = 0; i < count; i++) {
            this.addDice();
        }
        
        this.diceBodies.forEach(body => {
            body.velocity.set(
                (Math.random() - 0.5) * 5, 
                -15,  // Сильнее бросок вниз
                (Math.random() - 0.5) * 5
            );
            body.angularVelocity.set(
                Math.random() * 20, 
                Math.random() * 20, 
                Math.random() * 20
            );
        });
    }

    animate() {
        requestAnimationFrame(() => this.animate());

        this.world.step(1 / 60);

        for (let i = 0; i < this.diceMeshes.length; i++) {
            this.diceMeshes[i].position.copy(this.diceBodies[i].position);
            this.diceMeshes[i].quaternion.copy(this.diceBodies[i].quaternion);
        }

        // ВМЕСТО this.renderer.render ИСПОЛЬЗУЕМ composer
        this.composer.render();
    }

    onResize() {
        this.width = window.innerWidth;
        this.height = window.innerHeight;
        this.camera.aspect = this.width / this.height;
        this.camera.updateProjectionMatrix();
        this.renderer.setSize(this.width, this.height);
        this.composer.setSize(this.width, this.height);
        
        if (this.theme === 'industrial') {
            this.pixelPass.setPixelSize(3);
        } else {
            this.pixelPass.setPixelSize(1);
        }
    }
}