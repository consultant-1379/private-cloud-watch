from setuptools import setup, find_packages

setup(
    name='fm-alarm-lib',
    version='0.10',
    packages=find_packages(where="src"),
    package_dir={"": "src"},
    url='',
    license='',
    author='ekavosh',
    author_email='',
    description='',
    install_requires=['enum34', 'certifi', 'chardet', 'configparser', 'idna', 'python-consul', 'requests', 'six',
                      'urllib3'],
    zip_safe=False,
    entry_points={
        'console_scripts': ['fm-alarm-lib=executor.fm_alarm_library:main'],
    }
)
