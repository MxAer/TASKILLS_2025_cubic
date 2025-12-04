import * as THREE from 'three';
import * as CANNON from 'cannon-es';

export class DiceWorld {
    constructor(canvas) {
        this.canvas = canvas;
        this.width = window.innerWidth;
        this.height = window.innerHeight;
        this.theme = 'clean'; // 'clean' or 'industrial'

        // 1. Setup Three.js
        this.scene = new THREE.Scene();
        this.camera = new THREE.PerspectiveCamera(45, this.width / this.height, 0.1, 100);
        this.camera.position.set(0, 15, 10);
        this.camera.lookAt(0, 0, 0);

        this.renderer = new THREE.WebGLRenderer({ canvas: this.canvas, antialias: true, alpha: true });
        this.renderer.setSize(this.width, this.height);
        this.renderer.shadowMap.enabled = true;

        // 2. Setup Cannon.js (Physics)
        this.world = new CANNON.World();
        this.world.gravity.set(0, -9.82 * 5, 0); // Повышенная гравитация для веса
        this.world.broadphase = new CANNON.NaiveBroadphase();
        this.world.solver.iterations = 10;

        // Materials
        this.diceMaterialPhysics = new CANNON.Material();
        this.floorMaterialPhysics = new CANNON.Material();
        const contactMaterial = new CANNON.ContactMaterial(
            this.floorMaterialPhysics, this.diceMaterialPhysics,
            { friction: 0.01, restitution: 0.5 } // Скольжение и отскок
        );
        this.world.addContactMaterial(contactMaterial);

        // 3. Add Objects
        this.createFloor();
        this.createLights();

        // State
        this.diceMeshes = [];
        this.diceBodies = [];
        this.isAnimating = false;

        // Resize Listener
        window.addEventListener('resize', () => this.onResize());
    }

    createFloor() {
        // Visual Floor
        const geometry = new THREE.PlaneGeometry(100, 100);
        const material = new THREE.ShadowMaterial({ opacity: 0.3 });
        this.floorMesh = new THREE.Mesh(geometry, material);
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
        this.ambientLight = new THREE.AmbientLight(0xffffff, 0.5);
        this.scene.add(this.ambientLight);

        this.dirLight = new THREE.DirectionalLight(0xffffff, 1);
        this.dirLight.position.set(10, 20, 10);
        this.dirLight.castShadow = true;
        this.scene.add(this.dirLight);
        
        // SpotLight for Industrial Theme (hidden by default)
        this.spotLight = new THREE.SpotLight(0xea580c, 50);
        this.spotLight.position.set(0, 15, 0);
        this.spotLight.angle = Math.PI / 6;
        this.spotLight.penumbra = 1;
        this.spotLight.visible = false;
        this.scene.add(this.spotLight);
    }

    // Метод переключения темы
    setTheme(themeName) {
        this.theme = themeName;
        if (themeName === 'industrial') {
            this.scene.background = new THREE.Color(0x0c0a08);
            this.ambientLight.intensity = 0.1;
            this.dirLight.intensity = 0; // Выключаем солнце
            this.spotLight.visible = true; // Включаем "лампу"
        } else {
            this.scene.background = null; // Прозрачный (берет CSS body)
            this.ambientLight.intensity = 0.6;
            this.dirLight.intensity = 0.8;
            this.spotLight.visible = false;
        }
    }

    // Создание кубика (Пока D6)
    addDice() {
        // Visual
        const size = 1.5;
        const geometry = new THREE.BoxGeometry(size, size, size);
        
        // Цвет зависит от темы
        const color = this.theme === 'industrial' ? 0x78350f : 0x34d399;
        const material = new THREE.MeshStandardMaterial({ 
            color: color, 
            roughness: 0.2,
            metalness: 0.1
        });
        
        const mesh = new THREE.Mesh(geometry, material);
        mesh.castShadow = true;
        this.scene.add(mesh);
        this.diceMeshes.push(mesh);

        // Physics
        const shape = new CANNON.Box(new CANNON.Vec3(size/2, size/2, size/2));
        const body = new CANNON.Body({ mass: 1, material: this.diceMaterialPhysics });
        body.addShape(shape);
        
        // Стартовая позиция (случайно в воздухе)
        body.position.set((Math.random() - 0.5) * 5, 10, (Math.random() - 0.5) * 5);
        body.angularVelocity.set(Math.random() * 10, Math.random() * 10, Math.random() * 10);
        
        this.world.addBody(body);
        this.diceBodies.push(body);
    }

    clearDice() {
        // Удаляем старые кубики
        for(let mesh of this.diceMeshes) this.scene.remove(mesh);
        for(let body of this.diceBodies) this.world.removeBody(body);
        this.diceMeshes = [];
        this.diceBodies = [];
    }

    // Главный метод броска
    throwDice(count = 1) {
        this.clearDice();
        for (let i = 0; i < count; i++) {
            this.addDice();
        }
        
        // Импульс для всех тел
        this.diceBodies.forEach(body => {
            // Бросаем вниз и немного в сторону
            body.velocity.set(
                (Math.random() - 0.5) * 5, 
                -10, 
                (Math.random() - 0.5) * 5
            );
            // Закручиваем
            body.angularVelocity.set(
                Math.random() * 20, 
                Math.random() * 20, 
                Math.random() * 20
            );
        });
    }

    animate() {
        requestAnimationFrame(() => this.animate());

        // Physics Step
        this.world.step(1 / 60);

        // Sync Visuals with Physics
        for (let i = 0; i < this.diceMeshes.length; i++) {
            this.diceMeshes[i].position.copy(this.diceBodies[i].position);
            this.diceMeshes[i].quaternion.copy(this.diceBodies[i].quaternion);
        }

        this.renderer.render(this.scene, this.camera);
    }

    onResize() {
        this.width = window.innerWidth;
        this.height = window.innerHeight;
        this.camera.aspect = this.width / this.height;
        this.camera.updateProjectionMatrix();
        this.renderer.setSize(this.width, this.height);
    }
}
