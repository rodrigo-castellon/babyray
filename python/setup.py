from setuptools import setup, find_packages

setup(
    name='babyray',
    version='0.1.0',
    packages=find_packages(),
    install_requires=[
        'grpcio',  # Make sure all your dependencies are listed here
        'grpcio-tools'
    ],
    entry_points={
        'console_scripts': [
            'babyray-cli=babyray.client:main',  # If you have a CLI entry point
        ]
    }
)

